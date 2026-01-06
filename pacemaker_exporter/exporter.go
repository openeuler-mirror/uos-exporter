package main

import (
	"context"
	"fmt"
	"time"

	"pacemaker_exporter/internal/server"
	"pacemaker_exporter/pkg/logger"

	"github.com/sirupsen/logrus"
)

// Run initializes and runs the exporter server with proper error handling and graceful shutdown
func Run(name string, version string) error {
	// Initialize logging first
	logger.InitDefaultLog()

	// Create server instance
	s := server.NewServer(name, version)

	// Setup phase with structured error handling
	if err := setupServer(s); err != nil {
		return fmt.Errorf("server setup failed: %w", err)
	}

	// Run server in background with proper context and error handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go runServerWithErrorHandling(ctx, s, errChan)

	// Wait for exit signal or error
	select {
	case <-s.ExitSignal:
		logrus.WithFields(logrus.Fields{
			"server":  name,
			"version": version,
		}).Info("Received exit signal, shutting down gracefully")

		// Cancel context to stop server
		cancel()

		// Stop server with timeout
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()

		s.Stop()

		// Wait for server to finish or timeout
		select {
		case err := <-errChan:
			if err != nil {
				logrus.WithError(err).Error("Server stopped with error")
				return err
			}
		case <-stopCtx.Done():
			logrus.Warn("Server shutdown timeout")
			return fmt.Errorf("server shutdown timeout")
		}

		logrus.Info("Exporter server shutdown completed successfully")
		return nil

	case err := <-errChan:
		logrus.WithError(err).Error("Server failed unexpectedly")
		s.Exit()
		return fmt.Errorf("server runtime error: %w", err)
	}
}

// setupServer handles the server setup phase with proper error logging
func setupServer(s *server.Server) error {
	logrus.WithFields(logrus.Fields{
		"name":    s.Name,
		"version": s.Version,
	}).Info("Starting server setup")

	s.PrintVersion()

	if err := s.SetUp(); err != nil {
		logrus.WithError(err).Error("Server setup failed")
		return err
	}

	logrus.Info("Server setup completed successfully")
	return nil
}

// runServerWithErrorHandling runs the server with proper error handling and context support
func runServerWithErrorHandling(ctx context.Context, s *server.Server, errChan chan error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithField("panic", r).Error("Server panicked")
			errChan <- fmt.Errorf("server panic: %v", r)
		}
	}()

	logrus.WithField("server", s.Name).Info("Starting server")

	if err := s.Run(); err != nil {
		logrus.WithError(err).Error("Server run failed")
		s.Error = err
		errChan <- err
		return
	}

	// If we get here, server stopped normally
	errChan <- nil
	s.Exit()
}
