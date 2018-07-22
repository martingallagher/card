package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

var logger *zap.Logger

func main() {
	initLogger()

	var err error
	accounts, accountsMap, err = loadDB(dbFile)

	if err != nil {
		logger.Fatal("Failed to load accounts", zap.Error(err))
	}

	var addr string

	flag.StringVar(&addr, "a", "0.0.0.0:8080", "API address")

	r := chi.NewRouter()
	r.Get("/accounts", getAccounts)
	r.Post("/accounts", createAccount)
	r.Get("/accounts/{id}", getAccount)
	r.Get("/accounts/{id}/statement", statement)
	r.Post("/accounts/{id}/load", load)
	r.Post("/accounts/{id}/authorize", authorize)
	r.Post("/accounts/{id}/capture", capture)
	r.Post("/accounts/{id}/reverse", reverse)
	r.Post("/accounts/{id}/refund", refund)

	s := &http.Server{Addr: addr, Handler: r}

	go func() {
		logger.Info("Starting server", zap.String("address", addr))

		err := s.ListenAndServe()

		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to listen", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	<-stop

	logger.Info("Shutting down server")

	// Shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	s.Shutdown(ctx)

	logger.Info("Server gracefully stopped")
}

func initLogger() {
	var (
		err    error
		config zap.Config
	)

	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	config.DisableStacktrace = true
	logger, err = config.Build()

	if err != nil {
		log.Fatal(err)
	}
}
