package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/manas-solves/realworld-backend/internal/vcs"
)

var version = vcs.Version()

type envelope map[string]any

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := parseConfig()

	app := newApplication(cfg, logger)
	err := app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func parseConfig() appConfig {
	var cfg appConfig

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 50, "PostgreSQL max open connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")
	flag.DurationVar(&cfg.db.timeout, "db-timeout", 10*time.Second, "PostgreSQL operation timeout")

	flag.StringVar(&cfg.jwtMaker.secretKey, "jwt-secret", os.Getenv("JWT_SECRET"), "JWT secret key (minimum 32 characters)")
	flag.StringVar(&cfg.jwtMaker.issuer, "jwt-issuer", os.Getenv("JWT_ISSUER"), "JWT issuer")
	flag.DurationVar(&cfg.jwtMaker.accessDuration, "jwt-access-duration", 24*time.Hour, "JWT access token duration")

	// Create a new version boolean flag with the default value of false.
	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	return cfg
}
