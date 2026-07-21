package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"

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
	UUID         string
)

func Initialize() {
	InstanceName = viper.GetString("discovery.instanceName")
	ServiceName = viper.GetString("discovery.advanced.serviceName")
	Domain = viper.GetString("discovery.advanced.domain")
	Port = viper.GetInt("discovery.advanced.port")
	UUID = viper.GetString("discovery.advanced.deviceUUID")
}

type Device struct {
	DeviceName  string
	Address     string
	Status      int    // 0 and 1 corresponding to open-to-requests and busy (already sharing or DND)
	LastUpdated int    // time in seconds since last update
	UUID        string // unique permanent identifier for the device, does not change even if the device name changes
	Port        string // port on which the device is listening for incoming requests
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
	interfaces, err := getInterfaces()
	if err != nil {
		//ignore, just use nil
	}
	server, err := zeroconf.Register(
		InstanceName,
		ServiceName,
		Domain,
		Port,
		[]string{UUID, fmt.Sprintf("%d", viper.GetInt("webserver.port"))},
		interfaces,
	)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%s Broadcasting as %s (service: %s) \n", internal.Icons.Positive, InstanceName, ServiceName)

	return server
}

func ServiceBrowser() {
	interfaces, err := getInterfaces()
	if err != nil {
		//ignore, just use nil
	}
	resolver, err := zeroconf.NewResolver(zeroconf.SelectIfaces(interfaces))
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

			if len(entry.AddrIPv4) == 0 || len(entry.Text) < 2 {
				continue
			}

			// currently only using ipv4. if the port is not advertised then we omit the device outright
			Devices.Update(Device{
				DeviceName:  entry.Instance,
				Address:     entry.AddrIPv4[0].String(),
				Status:      0,
				LastUpdated: 0,
				UUID:        entry.Text[0],
				Port:        entry.Text[1], // the port on which the device is listening for incoming requests
			})

		}
	}(entries)

	err = resolver.Browse(context.Background(), ServiceName, Domain, entries)

	if err != nil {
		fmt.Println(err)
	}

}

func getInterfaces() ([]net.Interface, error) {
	allInterfaces, err := net.Interfaces()

	if err != nil {
		return nil, err
	}

	var usableInterfaces []net.Interface
	for _, entry := range allInterfaces {
		if entry.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if entry.Flags&net.FlagLoopback != 0 {
			continue // loopback interface (i have no clue what this means but was suggested so)
		}
		if entry.Flags&net.FlagMulticast == 0 {
			continue // no multicast support
		}

		addrs, err := entry.Addrs()

		if err != nil || len(addrs) == 0 {
			continue // no addresses associated with the interface
		}

		hasUsableAddr := false
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue // not an IP address
			}
			if ipNet.IP.IsLinkLocalUnicast() {
				continue // link-local address (not routable)
			}
			if ipNet.IP.To4() != nil {
				hasUsableAddr = true // found a usable IPv4 address
				break
			}
		}

		if hasUsableAddr {
			usableInterfaces = append(usableInterfaces, entry)
		}
	}

	return usableInterfaces, nil
}
