package discovery

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/nishitjha/drop/internal"
	"github.com/spf13/viper"
)

var InternalViper *viper.Viper
var internalViperMu sync.Mutex

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

	home, _ := os.UserHomeDir()

	InternalViper = viper.New()
	InternalViper.SetConfigType("json")
	InternalViper.SetConfigFile(filepath.Join(home, ".drop_known_devices.json"))
}

type Device struct {
	DeviceName   string
	Address      string
	Status       int       // 0 and 1 corresponding to active and inactive/busy resp.
	LastUpdated  int       // time in seconds since last update
	UUID         string    // unique permanent identifier for the device, does not change even if the device name changes
	Port         string    // port on which the device is listening for incoming requests
	LastSeenTime time.Time // timestamp of the last time the device was seen
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

func (cache *Cache) List() map[string]Device {
	cache.MuTex.RLock()
	defer cache.MuTex.RUnlock()

	devices := make(map[string]Device, len(cache.devices))
	for k, v := range cache.devices {
		devices[k] = v
	}
	return devices
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
				DeviceName:   entry.Instance,
				Address:      entry.AddrIPv4[0].String(),
				Status:       0,
				LastUpdated:  0,
				UUID:         entry.Text[0],
				Port:         entry.Text[1], // the port on which the device is listening for incoming requests'
				LastSeenTime: time.Now(),
			})

			// update in file cache
			updateStaleCache(Device{
				DeviceName:   entry.Instance,
				Address:      entry.AddrIPv4[0].String(),
				Status:       0,
				LastUpdated:  0,
				UUID:         entry.Text[0],
				Port:         entry.Text[1],
				LastSeenTime: time.Now(),
			})
		}
	}(entries)

	err = resolver.Browse(context.Background(), ServiceName, Domain, entries)

	if err != nil {
		fmt.Println(err)
	}

}

func RetrieveDevices() map[string]Device {
	// read from memory and file cache and merge the two. if a device is in memory, it takes precedence over the file cache obv
	// additionally, for a device in the file cache, if the time since last seen is more than 5 minutes:
	// we send a "are you alive my guy" request to the device and if it responds, we update the last seen time in the file cache and memory cache
	// if it doesn't respond, we remove it from the file cache and memory cache

	memDevices := Devices.List() // memdevices is of type map[string]Device where

	internalViperMu.Lock()
	defer internalViperMu.Unlock()

	if err := InternalViper.ReadInConfig(); err != nil {
		// handle error
		// NOT! haha im so funny
	}

	for _, deviceUUID := range InternalViper.AllKeys() {
		// check if any device in the memory devices list has the same UUID as the device in the file cache. if it does, we skip it
		if _, exists := memDevices[deviceUUID]; !exists {
			// device is in file cache but not in memory cache, check if it's stale
			var device Device
			if err := InternalViper.UnmarshalKey(deviceUUID, &device); err != nil {
				fmt.Println("unmarshal failed:", err) // temporary debug line
				continue
			}

			// if time.Since(device.LastSeenTime) > 5*time.Minute {
			// 	// to be implemented
			// 	InternalViper.Set(deviceUUID, nil)
			// 	continue
			// }

			Devices.Update(device)
			memDevices[deviceUUID] = device
		}
	}
	return memDevices
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

func updateStaleCache(device Device) {
	internalViperMu.Lock()
	defer internalViperMu.Unlock()

	if err := InternalViper.ReadInConfig(); err != nil {
		// handle error
		// NOT! haha im so funny
	}

	InternalViper.Set(device.UUID, device)
	err := InternalViper.WriteConfig()
	if err != nil {
		_ = InternalViper.SafeWriteConfig()
	}
}
