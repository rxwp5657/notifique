package middleware

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	ch "github.com/notifique/internal/cache"
	"github.com/notifique/internal/controllers"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func GetCache(cache ch.Cache, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		userId := c.GetHeader(controllers.UserIdHeaderKey)

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
}
