package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kevin/kkcert/internal/api"
	"github.com/kevin/kkcert/internal/scheduler"
	"github.com/kevin/kkcert/internal/store"
	"github.com/kevin/kkcert/internal/tz"
)

//go:embed all:dist
var frontendFS embed.FS

func main() {
	dataDir := envOr("KKCERT_DATA_DIR", "./data")
	listen := envOr("KKCERT_LISTEN", ":8080")

	flag.StringVar(&dataDir, "data", dataDir, "data directory")
	flag.StringVar(&listen, "listen", listen, "listen address")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	time.Local = tz.Location

	dbPath := dataDir + "/kkcert.db"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		slog.Error("create data dir", "err", err)
		os.Exit(1)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		slog.Error("open store", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	sched := scheduler.New(st, dataDir)
	if err := sched.Start(); err != nil {
		slog.Error("start scheduler", "err", err)
		os.Exit(1)
	}
	defer sched.Stop()

	distFS, _ := fs.Sub(frontendFS, "dist")
	srv := api.NewServer(st, sched, distFS)

	httpSrv := &http.Server{
		Addr:         listen,
		Handler:      srv,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", listen)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
