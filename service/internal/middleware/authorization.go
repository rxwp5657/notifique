package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/notifique/shared/auth"
)

func Authorize(requiredScopes ...auth.Scope) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		scopes := strings.Split(ctx.GetHeader(string(auth.ScopeHeader)), " ")

		if len(scopes) == 0 {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		scopeSet := make(map[string]struct{}, len(scopes))

		for _, scope := range scopes {
			scopeSet[scope] = struct{}{}
		}

		for _, requiredScope := range requiredScopes {
			if _, ok := scopeSet[string(requiredScope)]; !ok {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
		}

		ctx.Next()
	}
}
