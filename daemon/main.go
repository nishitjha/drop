package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kardianos/service"
	"github.com/nishitjha/drop/discovery"
	"github.com/nishitjha/drop/webserver"
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
		Arguments: func() []string {
			if runtime.GOOS == "windows" {
				return []string{"service", "win-start"}
			}
			return []string{"service", "internal-run"}
		}(),
	}

	d := &Daemon{
		stop: make(chan struct{}),
		run: func() {
			discovery.Initialize()
			discovery.LaunchService()
			discovery.ServiceBrowser()

			os.WriteFile(filepath.Join(os.TempDir(), "drop-config-debug.log"),
				[]byte(fmt.Sprintf("Instance=%q Service=%q Domain=%q Port=%d UUID=%q\n",
					discovery.InstanceName, discovery.ServiceName, discovery.Domain, discovery.Port, discovery.UUID)),
				0644)
			webserver.Listen("daemon")

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
		err = s.Stop()
		if err != nil {
			return err
		}
		err = s.Uninstall()
		if err != nil {
			return err
		}
		return nil

	case "internal-run":
		logger, _ := s.Logger(nil)
		fmt.Println("Running Drop service...")

		err = s.Run()
		if err != nil {
			logger.Error(err)
			return err
		}
		return nil

	case "win-start":
		d.run()
		return nil
	}
	return nil
}
