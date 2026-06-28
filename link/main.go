package link

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	fmt.Println("Exiting (not a timeout though)..")
}

func ServiceBrowser() {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			if entry.Instance == instanceName {
				continue //ignore your own device
			}

			log.Println("----------------------------------------")
			log.Printf("Found Remote Device: %s\n", entry.Instance)
			log.Printf("IP Address: %v\n", entry.AddrIPv4)
			log.Printf("Port: %d\n", entry.Port)
			log.Printf("Remote Message: %v\n", entry.Text)
			log.Println("----------------------------------------")
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = resolver.Browse(ctx, serviceName, domain, entries)

	if err != nil {
		log.Fatalln("Couldn't browse for services:", err.Error())
	}

	<-ctx.Done()

}
