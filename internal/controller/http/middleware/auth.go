package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/gliedabrennung/messenger-core/internal/common/api"
	"github.com/gliedabrennung/messenger-core/internal/common/authctx"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth(secret string) app.HandlerFunc {
	secretBytes := []byte(secret)
	return func(ctx context.Context, c *app.RequestContext) {
		var tokenStr string

		if t, ok := extractBearerToken(string(c.GetHeader("Authorization"))); ok {
			tokenStr = t
		} else if q := strings.TrimSpace(c.Query("token")); q != "" {
			tokenStr = q
		}

		if tokenStr == "" {
			api.ErrorResponse(c, http.StatusUnauthorized,
				"UNAUTHORIZED", "missing or malformed authorization header", nil)
			c.Abort()
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenUnverifiable
			}
			return secretBytes, nil
		})
		if err != nil || !token.Valid {
			api.ErrorResponse(c, http.StatusUnauthorized,
				"UNAUTHORIZED", "invalid or expired token", nil)
			c.Abort()
			return
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			api.ErrorResponse(c, http.StatusUnauthorized,
				"UNAUTHORIZED", "invalid subject claim", nil)
			c.Abort()
			return
		}

		authctx.SetUserID(c, userID)
		if claims.ExpiresAt != nil {
			authctx.SetTokenExp(c, claims.ExpiresAt.Time)
		}
		c.Next(ctx)
	}
}

func extractBearerToken(header string) (string, bool) {
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(header[7:])
	return token, token != ""
}
