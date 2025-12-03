package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	httpdelivery "rankr/app/delivery/http"
	userrepo "rankr/app/repository"
	"rankr/app/service/user"
	"rankr/pkg/database"
	"rankr/pkg/httpserver"
	"rankr/pkg/logger"
)

type Application struct {
	ShutdownCtx context.Context
	Repo        user.Repository
	Service     user.Service
	Handler     httpdelivery.Handler
	HTTPServer  httpdelivery.Server
	Config      Config
	Validator   user.Validator
}

func Setup(
	ctx context.Context,
	config Config,
	postgresConn *database.Database,
) (Application, error) {
	log := logger.L()

	repo := userrepo.NewUserRepository(postgresConn)
	validator := user.NewValidator()
	svc := user.NewService(repo, validator, user.ImportOptions{})

	httpSrvCore, err := httpserver.New(config.HTTPServer)
	if err != nil {
		log.Error("failed to initialize HTTP server", slog.Any("error", err))
		return Application{}, err
	}
	httpHandler := httpdelivery.NewHandler(svc)
	httpSrv := httpdelivery.New(*httpSrvCore, httpHandler)

	return Application{
		ShutdownCtx: ctx,
		Repo:        repo,
		Service:     svc,
		Handler:     httpHandler,
		HTTPServer:  httpSrv,
		Config:      config,
		Validator:   validator,
	}, nil
}

func (app Application) Start() {
	log := logger.L()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	startServers(app, &wg)

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownTimeoutCtx, cancel := context.WithTimeout(context.Background(), app.Config.TotalShutdownTimeout)
	defer cancel()

	if app.shutdownServers(shutdownTimeoutCtx) {
		log.Info("servers shut down gracefully")
	} else {
		log.Warn("shutdown timed out; forcing exit")
		os.Exit(1)
	}

	wg.Wait()
	log.Info("user_app stopped")
}

func startServers(app Application, wg *sync.WaitGroup) {
	log := logger.L()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("HTTP server starting", slog.Int("port", app.Config.HTTPServer.Port))
		if err := app.HTTPServer.Serve(); err != nil {
			log.Error("HTTP server error", slog.Int("port", app.Config.HTTPServer.Port), slog.Any("error", err))
		}
		log.Info("HTTP server stopped", slog.Int("port", app.Config.HTTPServer.Port))
	}()
}

func (app Application) shutdownServers(ctx context.Context) bool {
	log := logger.L()
	log.Info("starting userapp server shutdown process")

	shutdownDone := make(chan struct{})

	go func() {
		var shutdownWg sync.WaitGroup
		shutdownWg.Add(1)
		go app.shutdownHTTPServer(ctx, &shutdownWg)

		shutdownWg.Wait()
		close(shutdownDone)
		log.Info("HTTP server has been shut down successfully")
	}()

	select {
	case <-shutdownDone:
		return true
	case <-ctx.Done():
		return false
	}
}

func (app Application) shutdownHTTPServer(parentCtx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	log := logger.L()
	log.Info("starting graceful shutdown for HTTP server", slog.Int("port", app.Config.HTTPServer.Port))

	httpShutdownCtx, httpCancel := context.WithTimeout(parentCtx, app.Config.HTTPServer.ShutdownTimeout)
	defer httpCancel()

	if err := app.HTTPServer.Stop(httpShutdownCtx); err != nil {
		log.Error("HTTP server graceful shutdown failed", slog.Any("error", err))
		return
	}

	log.Info("HTTP server shut down successfully")
}
