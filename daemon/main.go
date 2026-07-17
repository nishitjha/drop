package daemon

import (
	"os/exec"

	"github.com/kardianos/service"
)

type Config struct {
	Dir  string
	Exec string
	Args []string
	Env  []string

	Stderr, Stdout string
}

type Daemon struct {
	stop chan struct{}
	// apparently using an empty struct is a good idea for this purpose because firstly it doesn't take up any space
	// and secondly the data itself is not important, we just need a signal to stop the daemon so ok gopher whatever you say

	service service.Service

	*Config
	cmd *exec.Cmd
}

func (d *Daemon) Start(s service.Service) error {
	// this function should not block the main thread, so we will run the actual daemon in a goroutine
	go d.cmd.Run()
	return nil
}

func (d *Daemon) Stop(s service.Service) error {
	return nil
}

func NewDaemon(config *Config) (*Daemon, error) {
	d := &Daemon{
		Config: config,
		stop:   make(chan struct{}),
		cmd:    exec.Command(config.Exec, config.Args...),
	}

	return d, nil
}
