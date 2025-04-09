package middleware

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/notifique/shared/auth"
	ch "github.com/notifique/shared/cache"
	redis "github.com/redis/go-redis/v9"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

type CacheConfigurator interface {
	GetTTL() (time.Duration, error)
}

type CacheCfg struct {
	Cache        ch.Cache
	Configurator CacheConfigurator
}

type CacheMiddleware gin.HandlerFunc

func NewCacheMiddleware(cfg CacheCfg) (CacheMiddleware, error) {

	ttl, err := cfg.Configurator.GetTTL()

	if err != nil {
		return nil, fmt.Errorf("failed to get cache ttl - %w", err)
	}

	cache := cfg.Cache

	handler := func(c *gin.Context) {

		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		userId := c.GetHeader(string(auth.UserHeader))

		key := ch.GetEntpointKey(c.Request.URL, &userId)
		cachedResp, err, ok := cache.Get(c.Request.Context(), key)

		slog.Info(fmt.Sprintf("key %s", string(key)))

		if err != nil {
			slog.Error(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}

		if ok {
			c.Data(http.StatusOK, "application/json", []byte(cachedResp))
			c.Abort()
			return
		}

		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}

		c.Writer = w

		c.Next()

		response := w.body.String()
		responseStatus := c.Writer.Status()

		if responseStatus != http.StatusOK {
			return
		}

		err = cache.Set(c.Request.Context(), key, response, ttl)

		if err != nil {
			slog.Error(fmt.Errorf("failed to cache response - %w", err).Error())
		}
	}

	return handler, nil
}

func NewRedisLimiter(client *redis.Client) *redis_rate.Limiter {
	return redis_rate.NewLimiter(client)
}
