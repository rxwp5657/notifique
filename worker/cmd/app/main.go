package main

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/notifique/shared/auth"
)

type Cognito struct {
	jwksURL string
}

func Authorize(requiredScopes ...auth.Scope) func(m map[string]string) bool {

	return func(m map[string]string) bool {
		scopes := strings.Split(m[string(auth.ScopeHeader)], " ")

		if len(scopes) == 0 {
			return false
		}

		scopeSet := make(map[string]struct{}, len(scopes))

		for _, scope := range scopes {
			scopeSet[scope] = struct{}{}
		}

		for _, requiredScope := range requiredScopes {
			if _, ok := scopeSet[string(requiredScope)]; !ok {
				return false
			}
		}

		return true
	}
}

func isValidToken(jwtToken string, c Cognito) bool {

	jwks, err := keyfunc.NewDefault([]string{c.jwksURL})

	if err != nil {
		err = fmt.Errorf("failed to create JWK Set from resource at the given URL: %w", err)
		slog.Error(err.Error())
		return false
	}

	token, err := jwt.Parse(jwtToken, jwks.Keyfunc)

	if err != nil {
		err = fmt.Errorf("failed to parse token: %w", err)
		slog.Error(err.Error())
		return false
	}

	if !token.Valid {
		slog.Error("The token is not valid.")
		return false
	}

	claims := token.Claims.(jwt.MapClaims)

	scopeMap := make(map[string]string)
	scopeMap[string(auth.ScopeHeader)] = claims["scope"].(string)

	fmt.Println("User:", claims["username"].(string))
	fmt.Println("Scope", claims["scope"].(string))
	fmt.Println("Perrito", claims["perrito"])

	authorizer := Authorize(auth.UserNotificationPublisher, auth.Scope("notifique/perrito"))

	if !authorizer(scopeMap) {
		slog.Error("The token does not have the required scopes.")
		return false
	}

	return true
}

func main() {

	// cfg := clients.CognitoAuthProviderCfg{
	// 	TokenUrl:     "http://cognito-idp.localhost.localstack.cloud:4566/_aws/cognito-idp/oauth2/token",
	// 	ClientID:     "widl0ts6cka45rvkot1yk41csp",
	// 	ClientSecret: "d709ac2a",
	// }

	// provider := clients.NewCognitoAuthProvider(cfg)

	// token, duration, err := provider.GenerateToken()

	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println("Token:", token)
	// fmt.Println("Duration:", duration.Second())

	token := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVhYjAyMjVhLTUzNDAtNDA2Zi1hMDRmLTY3OGRjMTI2ZWY4YyIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDQwOTQ5OTAsImlzcyI6Imh0dHA6Ly9sb2NhbGhvc3QubG9jYWxzdGFjay5jbG91ZDo0NTY2L3VzLWVhc3QtMV8yYzlkNTI2OTg5MzA0MDkyODdjN2JhZTdhMTY0OWQyYSIsInN1YiI6IndpZGwwdHM2Y2thNDVydmtvdDF5azQxY3NwIiwiYXV0aF90aW1lIjoxNzQ0MDkxMzkwLCJpYXQiOjE3NDQwOTEzOTAsImV2ZW50X2lkIjoiMGNjMmM2YWQtMjg2OC00MGE5LTk3MjgtYzExODg2NDYwNGEzIiwidG9rZW5fdXNlIjoiYWNjZXNzIiwidXNlcm5hbWUiOiJ3aWRsMHRzNmNrYTQ1cnZrb3QxeWs0MWNzcCIsImp0aSI6IjhkNGU5ZmNmLTdhOTMtNDQ2Mi1hYmZmLTM2OTc5NWQwYzRhNyIsImNsaWVudF9pZCI6IndpZGwwdHM2Y2thNDVydmtvdDF5azQxY3NwIiwic2NvcGUiOiJub3RpZmlxdWUvdXNlcl9ub3RpZmljYXRpb25fcHVibGlzaGVyIG5vdGlmaXF1ZS9zZWNvbmRfc2NvcGUifQ.aC3jO3590DSZjvnnFk6_fJLdEyfXYyyZ0J7tAYAOer1Q9CvPtIfxsHGX-11l3W_TC-qIN_gzWN4ym0P0loRscyGiQM2OYYneUJUEJMo_KYvSRVlAgXlPMdrKqmIJ6ZIgculhvOD0TBOpgwOSWiI2Z8k84-d4gf4KBLCVnorfCtrkJFfNF8DLrRChMTqIN3w-KvpOgVXgEv-zWgSUhRTEBxQJLEQUkdhZu2WWyQ3Fl-lY4ZwRj2yZMeXYSTLd17-wgPftgxAqCgZstQbjl93aGgi2MxgYDtRGLdiWuBI6qnXwNxToqCCXYRPYSYCAkJff3Unke5PMpkKEy9TcVoilDw"

	fmt.Println("Token:", token)
	fmt.Println("Is valid:", isValidToken(token, Cognito{jwksURL: "https://cognito-idp.localhost.localstack.cloud:4566/us-east-1_2c9d52698930409287c7bae7a1649d2a/.well-known/jwks.json"}))
}

//second_scope
