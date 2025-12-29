package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/yeliheng/go-ai-gateway/common/config"
	"github.com/yeliheng/go-ai-gateway/internal/cache"
)

type Limiter struct {
	Config config.RateLimitConfig
}

func NewLimiter() *Limiter {
	return &Limiter{
		Config: config.GlobalConfig.RateLimit,
	}
}

func (l *Limiter) Check(ctx context.Context, path string, method string, ip string, userID string) (bool, error) {
	if !l.Config.Enabled {
		return true, nil
	}

	rule := l.findRule(path, method)

	key := ""
	switch rule.Key {
	case "user_id":
		if userID == "" {
			key = "ratelimit:" + rule.Algo + ":" + path + ":ip:" + ip
		} else {
			key = "ratelimit:" + rule.Algo + ":" + path + ":user:" + userID
		}
	case "global":
		key = "ratelimit:" + rule.Algo + ":" + path + ":global"
	default: // "ip"
		key = "ratelimit:" + rule.Algo + ":" + path + ":ip:" + ip
	}

	switch rule.Algo {
	case "sliding_window":
		return l.checkSlidingWindow(ctx, key, rule)
	case "token_bucket":
		return l.checkTokenBucket(ctx, key, rule)
	default:
		// Default to token bucket if config error
		return l.checkTokenBucket(ctx, key, rule)
	}
}

func (l *Limiter) findRule(path string, method string) config.RuleConfig {
	for _, r := range l.Config.Rules {
		if r.Path == path && (r.Method == "" || r.Method == method) {
			return r
		}
	}
	return l.Config.Default
}

func (l *Limiter) checkTokenBucket(ctx context.Context, key string, rule config.RuleConfig) (bool, error) {
	rate := rule.Rate
	if rate <= 0 {
		rate = 1
	}
	burst := rule.Burst
	if burst <= 0 {
		burst = 1
	}

	now := float64(time.Now().UnixMilli()) / 1000.0 // Seconds

	// Execute Lua
	res, err := cache.RDB.Eval(ctx, resultTokenBucket, []string{key + ":tokens", key + ":ts"}, rate, burst, now, 1).Result()
	if err != nil {
		return true, fmt.Errorf("redis eval error: %w", err) // Fail open
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 1 {
		return true, nil
	}

	allowed := arr[0].(int64) == 1
	return allowed, nil
}

func (l *Limiter) checkSlidingWindow(ctx context.Context, key string, rule config.RuleConfig) (bool, error) {
	limit := rule.Limit
	if limit <= 0 {
		limit = 10
	}

	windowDuration, err := time.ParseDuration(rule.Window)
	if err != nil {
		windowDuration = time.Minute // Default fallback
	}
	windowMs := windowDuration.Milliseconds()
	nowMs := time.Now().UnixMilli()

	res, err := cache.RDB.Eval(ctx, resultSlidingWindow, []string{key}, windowMs, limit, nowMs).Result()
	if err != nil {
		return true, fmt.Errorf("redis eval error: %w", err)
	}

	allowed := res.(int64) == 1
	return allowed, nil
}
