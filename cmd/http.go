package cmd

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	exam "mono-test/feature/exam"
	leaderboard "mono-test/feature/leaderboard"
	"mono-test/feature/shared"
	"mono-test/feature/tryout"
	"mono-test/pkg"
	"net/http"
	"time"
)

func runHTTPServer(ctx context.Context) {
	// Load configuration
	cfg := shared.LoadConfig("config/app.yaml")
	grpcConn := pkg.InitGrpcConn()
	shutdownTracerProvider := pkg.InitTracerProvider(ctx, grpcConn)

	dbCfg, err := pgxpool.ParseConfig(cfg.DBConfig.ConnStr())
	if err != nil {
		log.Fatalln("unable to parse database config", err)
	}

	// Set needed dependencies
	pool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		log.Fatalln("unable to create database connection pool", err)
	}
	defer pool.Close()

	tryout.SetDBPool(pool)
	exam.SetDBPool(pool)
	exam.SetDBPool(pool)
	leaderboard.SetDBPool(pool)

	// Create a new server
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tryout.HttpRoute(mux)
	exam.HttpRoute(mux)
	leaderboard.HttpRoute(mux)

	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalln("unable to start server", err)
		}
	}()

	log.Println("server started")

	// Wait for signal to shut down
	<-ctx.Done()

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Fatalln("unable to shutdown server", err)
	}
	if err := shutdownTracerProvider(ctxShutDown); err != nil {
		log.Fatalf("failed to shutdown TracerProvider: %s", err)
	}

	log.Println("server shutdown")
}
