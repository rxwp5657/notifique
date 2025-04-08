package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/notifique/shared/auth"
)

type AuthConfigurator interface {
	GetJWKSURL() (string, error)
}

type AuthMiddleware gin.HandlerFunc

func NewAuthMiddleware(cfg AuthConfigurator) (AuthMiddleware, error) {

	jwksURL, err := cfg.GetJWKSURL()

	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS URL: %w", err)
	}

	handler := func(ctx *gin.Context) {

		authToken := ctx.GetHeader("Authorization")
		splitRes := strings.Split(authToken, "Bearer ")

		if authToken == "" || len(splitRes) != 2 {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		jwtToken := splitRes[1]
		jwks, err := keyfunc.NewDefault([]string{jwksURL})

		if err != nil {
			ctx.AbortWithStatus(http.StatusInternalServerError)
			err = fmt.Errorf("failed to create JWK Set from resource at the given URL: %w", err)
			slog.Error(err.Error())
			return
		}

		token, err := jwt.Parse(jwtToken, jwks.Keyfunc)

		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			err = fmt.Errorf("failed to parse token: %w", err)
			slog.Error(err.Error())
			return
		}

		if !token.Valid {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			slog.Error("The token is not valid.")
			return
		}

		claims := token.Claims.(jwt.MapClaims)

		if claims["username"] == nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			slog.Error("The token does not contain a username.")
			return
		}

		if claims["scope"] == nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			slog.Error("The token does not contain a scope.")
			return
		}

		ctx.Set(string(auth.UserHeader), claims["username"].(string))
		ctx.Set(string(auth.ScopeHeader), claims["scope"].(string))

		ctx.Next()
	}

	return handler, nil
}
