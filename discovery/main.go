package discovery

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"
)

var (
	InstanceName = "Nishits-Machine"
	ServiceName  = "_drop._tcp"
	domain       = "local."
	Port         = 3001
	metadata     = []string{"txtv=1", "message = i made poopy in my pants"}
)

type Device struct{
	DeviceName string
	Address string
	Status int // 0 and 1 corresponding to open-to-requests and busy (already sharing or DND)
	LastUpdated int // time in seconds since last update
	UUID string // unique permanent identifier for the device, does not change even if the device name changes
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

	cache.devices[device.UUID] = device
}

func (cache *Cache) List() (Devices map[string]Device){
	cache.MuTex.Lock()
	defer cache.MuTex.Unlock()

	return cache.devices
}

func LaunchService() {
	server, err := zeroconf.Register(
		InstanceName,
		ServiceName,
		domain,
		Port,
		metadata,
		nil, // auto-select from interfaces idk I'm not doing allat
	)

	if err != nil {
		panic(err)
	}

	defer server.Shutdown()

	fmt.Printf("Broadcasting as %[1]s (service: %[2]s) \n", InstanceName, ServiceName)

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

			if entry.Instance == InstanceName {
				continue //to ignore one's own broadcasts
			}

			// fmt.Println("------- OOGA - INSTANCE FOUND -------")
			// fmt.Printf("Device Name: %s\n", entry.Instance)
			// fmt.Printf("Address: %[1]v, %[2]v\n", entry.AddrIPv4, entry.AddrIPv6)
			// fmt.Println("------- OOGA -------")

			Devices.Update(Device{
				DeviceName: entry.Instance,
				Address: entry.AddrIPv4[0].String(),
				Status: 0,
				LastUpdated: 0,
				UUID: uuid.New().String(), //newly generated each time for now, but should be a permanent identifier for the device in the future
			})
		}
	}(entries)

	err = resolver.Browse(context.Background(), ServiceName, domain, entries)

	if err != nil {
		panic(err)
	}

}
