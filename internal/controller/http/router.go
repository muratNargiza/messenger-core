package http

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/gliedabrennung/messenger-core/internal/controller/http/middleware"
	"github.com/gliedabrennung/messenger-core/internal/pkg/api"
)

type Deps struct {
	Auth      AuthService
	WsHandler app.HandlerFunc
	JWTSecret string
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

	h.GET("/", ServeHome)

	auth := h.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	h.GET("/ws", middleware.JWTAuth(deps.JWTSecret), deps.WsHandler)
}
