package daemon

import (
	"github.com/kardianos/service"
)

type Daemon struct {
	stop chan struct{}
	// apparently using an empty struct is a good idea for this purpose because firstly it doesn't take up any space
	// and secondly the data itself is not important, we just need a signal to stop the daemon so ok gopher whatever you say

	service service.Service

	run func()
}

func (d *Daemon) Start(s service.Service) error {
	// this function should not block the main thread, so we will run the actual daemon in a goroutine
	d.service = s

	go d.run()
	return nil
}

func (d *Daemon) Stop(s service.Service) error {
	return nil
}

func Execute() error {
	svcConfig := &service.Config{
		Name:        "Drop",
		DisplayName: "Drop",
		Description: "Broadcasts and listens simultaneously in the background.",
	}

	d := &Daemon{
		stop: make(chan struct{}),
		run: func() {
			// run discovery.LaunchService() and discovery.ServiceBrowser() here
		},
	}

	s, err := service.New(d, svcConfig)
	if err != nil {
		return err
	}

	logger, err := s.Logger(nil) // idk if we need a logger but it seems like the best practice to have one
	if err != nil {
		return err
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}
