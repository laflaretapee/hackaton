package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if loc, e := time.LoadLocation(getenv("TZ", "Asia/Yekaterinburg")); e == nil {
		time.Local = loc
	}
	a, e := newApp(ctx, os.Getenv("DATABASE_URL"), os.Getenv("ADMIN_PASSWORD"), os.Getenv("ENCRYPTION_KEY"), getenv("APP_URL", "http://localhost:8080"), getenv("SECURE_COOKIES", "true") == "true")
	if e != nil {
		slog.Error("startup failed", "error", e)
		os.Exit(1)
	}
	defer a.db.Close()
	go a.botLoop(ctx)
	go a.scheduler(ctx)
	server := &http.Server{Addr: ":" + getenv("PORT", "8080"), Handler: a.routes(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 150 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	go func() {
		<-ctx.Done()
		stop, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = server.Shutdown(stop)
	}()
	slog.Info("AI-buddy API listening", "port", getenv("PORT", "8080"))
	if e = server.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
		slog.Error("http server failed", "error", e)
		os.Exit(1)
	}
}
