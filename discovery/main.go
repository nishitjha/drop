package discovery

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"
	"github.com/nishitjha/drop/internal"
	"github.com/spf13/viper"
)

var (
	InstanceName string
	ServiceName  string
	Domain       string
	Port         int
	Metadata     []string
	UUID string
)
func Initialize() {
	InstanceName = viper.GetString("discovery.instanceName")
	ServiceName  = viper.GetString("discovery.advanced.serviceName")
	Domain       = viper.GetString("discovery.advanced.domain")
	Port         = viper.GetInt("discovery.advanced.port")
	UUID = viper.GetString("discovery.advanced.deviceUUID")
}

type Device struct {
	DeviceName  string
	Address     string
	Status      int    // 0 and 1 corresponding to open-to-requests and busy (already sharing or DND)
	LastUpdated int    // time in seconds since last update
	UUID        string // unique permanent identifier for the device, does not change even if the device name changes
}

type Cache struct {
	MuTex   sync.RWMutex
	devices map[string]Device
}

// apparently you need a pointer cause creating a copy of a mutex is not a good idea
var Devices = &Cache{
	devices: make(map[string]Device),
}

func (cache *Cache) Update(device Device) {
	cache.MuTex.Lock()
	defer cache.MuTex.Unlock()

	cache.devices[device.UUID] = device
}

func (cache *Cache) List() (Devices map[string]Device) {
	cache.MuTex.Lock()
	defer cache.MuTex.Unlock()

	return cache.devices
}

func LaunchService() *zeroconf.Server {
	server, err := zeroconf.Register(
		InstanceName,
		ServiceName,
		Domain,
		Port,
		[]string{UUID},
		nil, // auto-select from interfaces idk I'm not doing allat
	)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%s Broadcasting as %s (service: %s) \n", internal.Icons.Positive, InstanceName, ServiceName)

	return server
}

func ServiceBrowser() {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		fmt.Println(err)
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
				DeviceName:  entry.Instance,
				Address:     entry.AddrIPv4[0].String(),
				Status:      0,
				LastUpdated: 0,
				UUID:        entry.Text[0], //newly generated each time for now, but should be a permanent identifier for the device in the future
			})

		}
	}(entries)

	//dummy devices for testing locally
	for i := 0; i < 4; i++ {
		Devices.Update(Device{
			DeviceName:  fmt.Sprintf("Test Device %d", i),
			Address:     fmt.Sprintf("192.168.0.%d", i+2),
			Status:      0,
			LastUpdated: 0,
			UUID:        uuid.New().String(),
		})
	}

	err = resolver.Browse(context.Background(), ServiceName, Domain, entries)

	if err != nil {
		fmt.Println(err)
	}

}