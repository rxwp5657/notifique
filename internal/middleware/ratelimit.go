package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/notifique/internal/controllers"
)

const (
	rateLimitLimit      string = "X-RateLimit-Limit"
	rateLimitRemaining  string = "X-RateLimit-Remaining"
	rateLimitReset      string = "X-RateLimit-Reset"
	rateLimitLimitRetry string = "X-RateLimit-Retry"
)

func RateLimit(limiter *redis_rate.Limiter, persecond int) gin.HandlerFunc {
	return func(c *gin.Context) {

		userId := c.GetHeader(controllers.UserIdHeaderKey)

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
}
