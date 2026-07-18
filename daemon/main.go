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

// getting a little messy here so I'll exlplain it to future me here
// windows does not allow services that are spawned programatically to actually interact w the desktop
// so even though you can launch a service and have it broadcast and whatnot, that service in itself will not be able to convey anything to the user if a request comes through
// i don't think unix has this problem but then again i haven't tested it yet
// a solution I found was to use the windows task scheduler to launch the daemon directly in the background
// note that this does not make use of kardianos/service (on windows)
// win-start (used by schtasks) runs the daemon directly in the background and can interact with the desktop
// internal-run is for unix systems where kardianos/service is used to make it a proper service
// all other commands are user-facing

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

	psScript := fmt.Sprintf(`
$action = New-ScheduledTaskAction -Execute '%s' -Argument 'service win-start'
$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -DontStopOnIdleEnd -RestartOnIdle -MultipleInstances IgnoreNew -ExecutionTimeLimit (New-TimeSpan -Seconds 0) -DisallowHardTerminate
Register-ScheduledTask -TaskName 'Drop' -Action $action -Trigger $trigger -Settings $settings -Force
`, dropwPath)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	if err := cmd.Run(); err != nil {
		fmt.Printf("%s Failed to create scheduled task: %v\n", internal.Icons.Negative, cmd)
		return err
	}
	return exec.Command("schtasks", "/run", "/tn", "Drop").Run()
}

func Execute(action string) error {
	if runtime.GOOS == "windows" {
		switch action {
		case "install":
			return windowsInstall()
		case "start":
			return exec.Command("schtasks", "/run", "/tn", "Drop").Run()
		case "kill":
			return exec.Command("schtasks", "/end", "/tn", "Drop").Run()
		case "uninstall":
			exec.Command("schtasks", "/end", "/tn", "Drop").Run()
			return exec.Command("schtasks", "/delete", "/tn", "Drop", "/f").Run()
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
		Description: "i made poopy in my pants",
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
		fmt.Println("Running Drop...")

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
		defer func() {
			if r := recover(); r != nil {
				os.WriteFile(filepath.Join(os.TempDir(), "drop-panic-debug.log"),
					[]byte(fmt.Sprintf("panic: %v\n", r)), 0644)
			}
		}()
		discovery.Initialize()
		discovery.LaunchService()
		go discovery.ServiceBrowser()

		os.WriteFile(filepath.Join(os.TempDir(), "drop-panic-debug.log"),
			[]byte(fmt.Sprintf("stuff")), 0644)
		webserver.Listen("daemon")
	}
}
