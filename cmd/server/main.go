package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/gliedabrennung/messenger-core/internal/config"
	"github.com/gliedabrennung/messenger-core/internal/controller/http"
	"github.com/gliedabrennung/messenger-core/internal/messenger"
	"github.com/gliedabrennung/messenger-core/internal/repository/postgres"
	"github.com/gliedabrennung/messenger-core/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("startup failed: %v", err)
	}
}

func run() error {
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx := context.Background()

	dbpool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return fmt.Errorf("create connection pool: %w", err)
	}
	defer dbpool.Close()

	if sqlBytes, err := os.ReadFile("migrations/0001.sql"); err == nil {
		if _, err := dbpool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}
	} else {
		log.Printf("WARNING: could not read migration file: %v", err)
	}

	repo := postgres.NewPostgresRepository(dbpool)
	authUseCase := usecase.NewAuthUseCase(repo, cfg.JWTSecret, cfg.JWTTTL)

	hub := messenger.NewHub()
	go hub.Run()

	h := server.Default(
		server.WithHostPorts(cfg.Addr),
		server.WithHandleMethodNotAllowed(true),
	)

	h.OnShutdown = append(h.OnShutdown, func(ctx context.Context) {
		hub.Stop()
	})

	upgrader := messenger.NewUpgrader(cfg.AllowedOrigins)
	wsHandler := messenger.ServeWs(hub, upgrader)

	http.SetupRouter(h, http.Deps{
		Auth:      authUseCase,
		WsHandler: wsHandler,
		JWTSecret: cfg.JWTSecret,
	})

	h.Spin()
	return nil
}
