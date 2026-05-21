package http

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/gliedabrennung/messenger-core/internal/common/api"
	"github.com/gliedabrennung/messenger-core/internal/controller/http/middleware"
)

type Deps struct {
	Auth           AuthService
	WsHandler      app.HandlerFunc
	AuthMiddleware app.HandlerFunc
}

func SetupRouter(h *server.Hertz, deps Deps) {
	h.Use(api.CustomErrorHandler())
	h.NoRoute(func(ctx context.Context, c *app.RequestContext) {
		api.ErrorResponse(c, http.StatusNotFound,
			"NOT_FOUND",
			"Page not found",
			nil)
	})
	h.NoMethod(func(ctx context.Context, c *app.RequestContext) {
		api.ErrorResponse(c, http.StatusMethodNotAllowed,
			"METHOD_NOT_ALLOWED",
			"Method not allowed",
			nil)
	})

	authHandler := NewAuthHandler(deps.Auth)
	authLimiter := middleware.NewRateLimiter(5, 10)

	h.GET("/", ServeHome)

	auth := h.Group("/auth")
	auth.Use(authLimiter.Handler())
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	h.GET("/ws", deps.AuthMiddleware, deps.WsHandler)
}
