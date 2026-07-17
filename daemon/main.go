package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/kardianos/service"
	"github.com/nishitjha/drop/discovery"
	"github.com/nishitjha/drop/internal"
	"github.com/nishitjha/drop/webserver"
)

type Daemon struct {
	stop chan struct{}

	service service.Service

	run func()
}

func (d *Daemon) Start(s service.Service) error {
	d.service = s
	go d.run()

	return nil
}

func (d *Daemon) Stop(s service.Service) error {
	close(d.stop)
	return nil
}

const windowsTaskName = "Drop"

func windowsInstall() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	dropwPath := filepath.Join(filepath.Dir(exePath), "dropw.exe")
	if _, err := os.Stat(dropwPath); err != nil {
		fmt.Printf("%s dropw.exe not found next to drop.exe. You probably forgot to build it.", internal.Icons.Negative)
		return err
	}

	createCmd := exec.Command("schtasks", "/create", "/tn", windowsTaskName,
		"/tr", fmt.Sprintf(`"%s" service install`, dropwPath),
		"/sc", "onlogon", "/f")
	if err := createCmd.Run(); err != nil {
		fmt.Printf("%s Failed to create scheduled task: %v\n", internal.Icons.Negative, createCmd)
		return err
	}

	return exec.Command("schtasks", "/run", "/tn", windowsTaskName).Run()
}

func windowsUninstall() error {
	return exec.Command("schtasks", "/delete", "/tn", windowsTaskName, "/f").Run()
}

func windowsKill() error {
	return exec.Command("schtasks", "/end", "/tn", windowsTaskName).Run()
}

func Execute(action string) error {
	if runtime.GOOS == "windows" {
		switch action {
		case "install":
			return windowsInstall()
		case "start":
			return exec.Command("schtasks", "/run", "/tn", windowsTaskName).Run()
		case "kill":
			return windowsKill()
		case "uninstall":
			windowsKill()
			return windowsUninstall()
		case "win-start":
			d := &Daemon{
				stop: make(chan struct{}),
				run:  runFunc(),
			}
			d.run()
			return nil
		}
		return nil
	}

	svcConfig := &service.Config{
		Name:        "Drop",
		DisplayName: "Drop",
		Description: "Broadcasts and listens simultaneously in the background.",
		Arguments:   []string{"service", "internal-run"},
		Option: service.KeyValue{
			"UserService": true,
		},
	}

	d := &Daemon{
		stop: make(chan struct{}),
		run:  runFunc(),
	}

	s, err := service.New(d, svcConfig)
	if err != nil {
		return err
	}

	switch action {
	case "install":
		if err := s.Install(); err != nil {
			return err
		}
		return s.Start()

	case "start":
		return s.Start()

	case "kill":
		return s.Stop()

	case "uninstall":
		s.Stop()
		return s.Uninstall()

	case "internal-run":
		logger, _ := s.Logger(nil)
		fmt.Println("Running Drop service...")

		if err := s.Run(); err != nil {
			logger.Error(err)
			return err
		}
		return nil
	}
	return nil
}

func runFunc() func() {
	return func() {
		discovery.Initialize()
		discovery.LaunchService()
		go discovery.ServiceBrowser()

		webserver.Listen("daemon")
	}
}
