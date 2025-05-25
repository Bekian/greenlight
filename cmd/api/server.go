package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// DIFF Note: var server is "srv" in the book
func (app *application) serve() error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	// channel to receive any errors during shutdown function
	shutdownError := make(chan error)

	// run a background goroutine to listen for signals
	go func() {
		// create a quit channel to hold signals
		quit := make(chan os.Signal, 1)
		// listen for provided signals
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		// block the channel until a signal is received and read the signal.
		s := <-quit

		// log caught signal
		app.logger.Info("shutting down server", "signal", s.String())

		// context with timeout, this allows 30s for any outstanding responses.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// call shutdown with the context
		shutdownError <- server.Shutdown(ctx)
	}()

	// display server start
	app.logger.Info("starting server", "addr", server.Addr, "env", app.config.env)

	err := server.ListenAndServe()
	// check if error is NOT the following, we want this error to happen
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// receive err value
	err = <-shutdownError
	if err != nil {
		return err
	}

	// log success
	app.logger.Info("stopped server", "addr", server.Addr)
	return nil
}
