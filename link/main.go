package link

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/grandcat/zeroconf"
)

var (
	instanceName = "Nishits-Laptop"
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

	fmt.Printf("Broadcasting as %[1]s (service: %[2]s) \n", instanceName, serviceName)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("Exiting..")
}

func ServiceBrowser() {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		panic(err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {

			if entry.Instance == instanceName {
				continue //to ignore one's own broadcasts
			}

			fmt.Println("------- OOGA - INSTANCE FOUND -------")
			fmt.Printf("Device Name: %s\n", entry.Instance)
			fmt.Printf("IP Address: %v\n", entry.AddrIPv4)
			fmt.Println("------- OOGA -------")

			//may choose to print out other attributes later
			//gonna cache instances and their static addresses as I discover them
			//protcol coming soon!!!111!
		}
	}(entries)

	err = resolver.Browse(context.Background(), serviceName, domain, entries)

	if err != nil {
		panic(err)
	}

}
