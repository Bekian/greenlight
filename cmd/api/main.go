package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Bekian/greenlight/internal/data"
	"github.com/Bekian/greenlight/internal/mailer"
	"github.com/Bekian/greenlight/internal/vcs"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// auto generated version number
var (
	version = vcs.Version()
)

// config options struct, this will use cli flags later
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     float64 // requests per second
		burst   int     // maximum amount of requests a user can make at once
		enabled bool    // flag to enable/disable rate-limiting
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

// app struct for dep injection across the app
type application struct {
	config config
	logger *slog.Logger
	models data.Models
	mailer *mailer.Mailer
	wg     sync.WaitGroup
}

// DIFF Note: several CLI flag default values use local environment variables for security.
func main() {
	// declare an instance of config to use in our app
	var cfg config

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// read the cli flags into the config struct
	// DIIF Note: the 8080 port deviates from the 4000 port used in book
	flag.IntVar(&cfg.port, "port", 8080, "API server port")
	// the "testing" value deviates from "staging", as "testing" is more common here
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|testing|production)")
	// read dsn value cli flag, or use default when none are provided
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	// set db config settings (pasted for safety)
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")
	// flags for rate limiting
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	// flags for mail server
	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 587, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", os.Getenv("SMTP_SENDER"), "SMTP sender")

	// use flag.func to read origins arg
	// DIFF Note: var s is "val"
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(s string) error {
		cfg.cors.trustedOrigins = strings.Fields(s)
		return nil
	})

	// flag to display version number and exit
	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	// use display version flag
	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	// init logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// init db by opening with cfg using helper (see below)
	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer db.Close()

	// log success
	logger.Info("db connection pool established")

	// init mailer instance
	mailer, err := mailer.New(
		cfg.smtp.host,
		cfg.smtp.port,
		cfg.smtp.username,
		cfg.smtp.password,
		cfg.smtp.sender,
	)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// publish version number to metrics
	expvar.NewString("version").Set(version)
	// publish number of active goroutines
	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
	// publish database conneciton pool stats
	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))
	// publish current unix timestamp
	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	// declare app object and pass in it's properties
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer,
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	// use sql open to make an empty connection pool
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// use the settings we created using db flags
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	// create context with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ping context creates a connection to the db
	// throws error if not within 5 second timeout
	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	// success
	return db, nil
}
