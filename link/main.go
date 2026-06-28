package link

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/grandcat/zeroconf"
)

var (
	instanceName = "Nishits_Machine"
	serviceName  = "_drop._tcp"
	domain       = "local."
	port         = 3001
	metadata     = []string{"txtv=1", "message = i made poopy in my pants"}
)

func LaunchService() {
	server, err := zeroconf.Register(
		instanceName,
		serviceName,
		domain,
		port,
		metadata,
		nil, // auto-select from interfaces idk I'm not doing allat
	)

	if err != nil {
		panic(err)
	}

	defer server.Shutdown()

	fmt.Printf("Broadcasting as %[1]s (service: %[2]s)", instanceName, serviceName)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("Exiting (not a timeout though)..")
}
