package handler

import (
	"ai-gateway/api/gen/identity/v1"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	identityClient identityv1.IdentityServiceClient
}

func NewAuthHandler(client identityv1.IdentityServiceClient) *AuthHandler {
	return &AuthHandler{
		identityClient: client,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.identityClient.Register(c.Request.Context(), &identityv1.RegisterRequest{
		Username: input.Username,
		Password: input.Password,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": resp.Message, "user_id": resp.UserId})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.identityClient.Login(c.Request.Context(), &identityv1.LoginRequest{
		Username: input.Username,
		Password: input.Password,
	})

	if err != nil {
		// Map gRPC errors to HTTP status if needed, simplified here
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    resp.Token,
		"username": resp.Username,
		"role":     resp.Role,
	})
}
