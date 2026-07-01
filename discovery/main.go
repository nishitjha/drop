package discovery

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/grandcat/zeroconf"
)

var (
	instanceName = "Nishits-Machine"
	serviceName  = "_drop._tcp"
	domain       = "local."
	port         = 3001
	metadata     = []string{"txtv=1", "message = i made poopy in my pants"}
)

type Device struct{
	deviceName string
	ipV4 string
	status int // 0 and 1 corresponding to open-to-requests and busy (already sharing or DND)
	lastUpdated int // time in seconds since last update
	uuid string // unique permanent identifier for the device, does not change even if the device name changes
}

type Cache struct{
	MuTex sync.RWMutex
	devices map[string]Device
}

// apparently you need a pointer cause creating a copy of a mutex is not a good idea
var Devices = &Cache{
	devices: make(map[string]Device),
}

func (cache *Cache) Update(device Device){
	cache.MuTex.Lock()
	defer cache.MuTex.Unlock()

	cache.devices[device.uuid] = device
}

func (cache *Cache) List() (Devices map[string]Device){
	cache.MuTex.Lock()
	defer cache.MuTex.Unlock()

	return cache.devices
}

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
			fmt.Printf("IPv4 Address: %v\n", entry.AddrIPv4)
			fmt.Printf("IPv6 Address: %v\n", entry.AddrIPv6)
			fmt.Println("------- OOGA -------")


		}
	}(entries)

	err = resolver.Browse(context.Background(), serviceName, domain, entries)

	if err != nil {
		panic(err)
	}

}
