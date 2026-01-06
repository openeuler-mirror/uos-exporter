package main

import (
	"nginx_vts_exporter/internal/server"
	"nginx_vts_exporter/pkg/logger"

	"github.com/sirupsen/logrus"
)

func Run(name string, version string) error {
	logger.InitDefaultLog()
	s := server.NewServer(name, version)

	s.PrintVersion()
	err := s.SetUp()
	if err != nil {
		logrus.Errorf("SetUp error: %v", err)
		return err
	}
	go func() {
		err := s.Run()
		if err != nil {
			logrus.Errorf("Run error: %v", err)
			s.Error = err
		}

		s.Exit()
	}()
	select {
	case <-s.ExitSignal:
		s.Stop()
		logrus.Info("Exit exporter server completed")
		return s.Error
	}
}
// Part 2 commit for nginx_vts_exporter/exporter.go
