package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig
	Services  ServicesConfig
	Auth      AuthConfig
	OpenAI    OpenAIConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	RateLimit RateLimitConfig
}

type ServicesConfig struct {
	Gateway  ServiceConfig
	Identity ServiceConfig
	Agent    ServiceConfig
	Biz      ServiceConfig
	Jaeger   ServiceConfig
}

type ServiceConfig struct {
	Name string
	Port string
	Addr string
}

type RateLimitConfig struct {
	Enabled bool
	Default RuleConfig
	Rules   []RuleConfig
}

type RuleConfig struct {
	Path   string
	Method string
	Algo   string // "token_bucket", "sliding_window"
	Key    string // "ip", "user_id", "global"
	// Token Bucket params
	Rate  float64
	Burst int
	// Sliding Window params
	Limit  int
	Window string // duration string
}

type AppConfig struct {
	Name string
	Port string
}

type AuthConfig struct {
	Tokens []string
}

type OpenAIConfig struct {
	ApiToken     string
	BaseURL      string
	Model        string
	SystemPrompt string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	ExpiryDays int
}

var GlobalConfig Config

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; ignore error if we rely on Env vars
	}

	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		return fmt.Errorf("unable to decode into struct: %w", err)
	}

	return nil
}
