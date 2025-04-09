package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/notifique/shared/auth"
)

const (
	rateLimitLimit      string = "X-RateLimit-Limit"
	rateLimitRemaining  string = "X-RateLimit-Remaining"
	rateLimitReset      string = "X-RateLimit-Reset"
	rateLimitLimitRetry string = "X-RateLimit-Retry"
)

type RateLimitConfigurator interface {
	GetRequestsPerSecond() (int, error)
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error)
}

type RateLimitCfg struct {
	RateLimiter  RateLimiter
	Configurator RateLimitConfigurator
}

type RateLimitMiddleware gin.HandlerFunc

func NewRateLimitMiddleware(cfg RateLimitCfg) (RateLimitMiddleware, error) {

	persecond, err := cfg.Configurator.GetRequestsPerSecond()

	if err != nil {
		return nil, err
	}

	limiter := cfg.RateLimiter

	handler := func(c *gin.Context) {

		userId := c.GetHeader(string(auth.UserHeader))

		if userId == "" {
			slog.Info("Unauthorized")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		res, err := limiter.Allow(c.Request.Context(), userId, redis_rate.PerSecond(persecond))

		if err != nil {
			slog.Error(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			c.Abort()
			return
		}

		if res.Allowed == 0 {
			slog.Info(fmt.Sprintf("Rate limit exceeded for user %s", userId))
			c.Header(rateLimitLimitRetry, strconv.FormatInt(res.RetryAfter.Nanoseconds()/1e6, 10))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too Many Requests"})
			c.Abort()
			return
		}

		c.Header(rateLimitLimit, strconv.FormatInt(int64(persecond), 10))
		c.Header(rateLimitRemaining, strconv.FormatInt(int64(res.Remaining), 10))
		c.Header(rateLimitReset, strconv.FormatInt(res.RetryAfter.Nanoseconds()/1e6, 10))

		c.Next()
	}

	return handler, nil
}
