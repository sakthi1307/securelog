package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sakthi1307/securelog/internal/config"
	"github.com/sakthi1307/securelog/internal/httpserver"
	"github.com/sakthi1307/securelog/internal/store"
	"github.com/sakthi1307/securelog/internal/rules"

)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := store.NewPool(ctx, cfg.DatabaseURL)

	ruleQueue := make(chan rules.RuleEvalMsg, 10000)
	alerts := store.NewAlertStore(pool)

	engine := &rules.Engine{
		DB:     pool,
		Alerts: alerts,
		Queue:  ruleQueue,
	}

	go engine.Run(ctx)
	defer close(ruleQueue)


	if err != nil {
		log.Error("db_connect_failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpserver.NewRouter(httpserver.Deps{
			Log: log, APIKey: cfg.APIKey, DB: pool, RuleQueue: ruleQueue,
		}),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server_start", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server_error", "err", err)
			stop()
		}
	}()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("server_stop")
}
