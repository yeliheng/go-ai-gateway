package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yeliheng/go-ai-gateway/api/gen/identity/v1"
	"github.com/yeliheng/go-ai-gateway/common/config"
	"github.com/yeliheng/go-ai-gateway/common/model"
	"github.com/yeliheng/go-ai-gateway/internal/cache"
	"github.com/yeliheng/go-ai-gateway/internal/database"

	"github.com/yeliheng/go-ai-gateway/common/logger"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Server struct {
	identityv1.UnimplementedIdentityServiceServer
}

func NewIdentityServer() *Server {
	return &Server{}
}

func (s *Server) Register(ctx context.Context, req *identityv1.RegisterRequest) (*identityv1.RegisterResponse, error) {
	logger.Log.Info("Register request received", zap.String("username", req.Username))
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Log.Error("Failed to hash password", zap.Error(err))
		return nil, err
	}

	// Default role: user
	var role model.Role
	if err := database.DB.FirstOrCreate(&role, model.Role{Name: "user"}).Error; err != nil {
		logger.Log.Error("Failed to create role", zap.Error(err))
		return nil, err
	}

	user := model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		RoleID:   role.ID,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		logger.Log.Error("Failed to create user", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("User registered successfully", zap.Uint("user_id", user.ID))
	return &identityv1.RegisterResponse{
		UserId:  fmt.Sprint(user.ID),
		Message: "User created successfully",
	}, nil
}

func (s *Server) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	logger.Log.Info("Login request received", zap.String("username", req.Username))
	var user model.User
	if err := database.DB.Preload("Role").Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Log.Warn("Login failed: user not found", zap.String("username", req.Username))
			return nil, errors.New("invalid credentials")
		}
		logger.Log.Error("Login failed: db error", zap.Error(err))
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	expiry := time.Duration(config.GlobalConfig.JWT.ExpiryDays) * 24 * time.Hour
	expiresAt := time.Now().Add(expiry)
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role.Name,
		"exp":  expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.GlobalConfig.JWT.Secret))
	if err != nil {
		return nil, err
	}

	err = cache.SetToken(ctx, tokenString, user.ID, expiry)
	if err != nil {
		return nil, err
	}

	return &identityv1.LoginResponse{
		Token:     tokenString,
		Username:  user.Username,
		Role:      user.Role.Name,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *identityv1.ValidateTokenRequest) (*identityv1.ValidateTokenResponse, error) {
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GlobalConfig.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		return &identityv1.ValidateTokenResponse{Valid: false}, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return &identityv1.ValidateTokenResponse{Valid: false}, nil
	}

	userID, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)

	cachedUserID, err := cache.GetToken(ctx, req.Token)
	if err != nil || cachedUserID != userID {
		return &identityv1.ValidateTokenResponse{Valid: false}, nil
	}

	return &identityv1.ValidateTokenResponse{
		Valid:    true,
		UserId:   userID,
		Username: "", // Context doesn't strictly need username here, but we could fetch it if needed
		Role:     role,
	}, nil
}
