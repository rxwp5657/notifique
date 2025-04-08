package mocks

import (
	"github.com/gin-gonic/gin"
	"github.com/notifique/service/internal/middleware"
	"github.com/notifique/shared/auth"
)

func NewTestAuthMiddleware() middleware.AuthMiddleware {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}

func NewTestCacheMiddleware() middleware.CacheMiddleware {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}

func NewTestSecurityMiddleware() middleware.SecurityMiddleware {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}

func NewTestRateLimitMiddleware() middleware.RateLimitMiddleware {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}

func TestAuthorize(requiredScopes ...auth.Scope) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
	}
}
