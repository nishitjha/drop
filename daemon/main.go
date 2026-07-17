package daemon

import (
	"fmt"

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
	close(d.stop)
	return nil
}

func Execute(action string) error {
	svcConfig := &service.Config{
		Name:        "Drop",
		DisplayName: "Drop",
		Description: "Broadcasts and listens simultaneously in the background.",
		Arguments:   []string{"service", "run"},
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

	switch action {
	case "install":
		err = s.Install()
		if err != nil {
			return err
		}
		err = s.Start()
		if err != nil {
			return err
		}
		return nil

	case "start":
		err = s.Start()
		if err != nil {
			return err
		}
		return nil

	case "kill":
		err = s.Stop()
		if err != nil {
			return err
		}
		return nil

	case "uninstall":
		err = s.Uninstall()
		if err != nil {
			return err
		}
		return nil

	case "run":
		logger, _ := s.Logger(nil)
		fmt.Println("Running Drop service...")

		err = s.Run()
		if err != nil {
			logger.Error(err)
			return err
		}
		return nil

	}
	return nil
}
