package devices

import (
	"fmt"

	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func LinkBLE() {
	err := adapter.Enable()
	if err != nil {
		panic(err)
	}

	serviceUUID, _ := bluetooth.ParseUUID("12345678-1234-5678-1234-567812345678")
	charUUID, _ := bluetooth.ParseUUID("87654321-4321-8765-4321-876543218765")

	adapter.AddService(&bluetooth.Service{
		UUID: serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				UUID:  charUUID,
				Value: []byte("192.168.1.15:8080"),
				Flags: bluetooth.CharacteristicReadPermission,
			},
		},
	})

	// fmt.Println("Scanning for BLE central..")
	// err_ := adapter.Scan( func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
	// 	println("found device:", device.Address.String(), device.RSSI, device.LocalName())
	// })

	// if err_ != nil {
	// 	panic(err_)
	// }

	advert := adapter.DefaultAdvertisement()
	fmt.Println("Advertising")

	advert.Configure(bluetooth.AdvertisementOptions{
		LocalName: "Nishit's Machine",
	})

	advert.Start()
}



// func Link(){
	
// // 1. Grab all network interfaces on your PC
// interfaces, err := net.Interfaces()
// if err != nil {
//     log.Fatalln("Failed to get network interfaces:", err)
// }

// var activeInterfaces []net.Interface
// for _, ifi := range interfaces {
//     // Only use active, non-loopback, multicast-capable interfaces
//     if (ifi.Flags&net.FlagUp) != 0 && 
//        (ifi.Flags&net.FlagLoopback) == 0 && 
//        (ifi.Flags&net.FlagMulticast) != 0 {
//         activeInterfaces = append(activeInterfaces, ifi)
//         // Optional: Print this to see which adapters it found
//         // log.Println("Using adapter:", ifi.Name) 
//     }
// }
// 	fmt.Println("Initializing..")
// resolver, err := zeroconf.NewResolver(zeroconf.SelectIfaces(activeInterfaces))
// if err != nil {
//     log.Fatalln("Failed to initialize resolver:", err)
// }

// entries := make(chan *zeroconf.ServiceEntry)
// go func(results <-chan *zeroconf.ServiceEntry) {
//     for entry := range results {
//         log.Println(entry)
//     }
//     log.Println("No more entries.")
// }(entries)

// ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
// defer cancel()
// err = resolver.Browse(ctx, "_spotify-connect._tcp", "local.", entries)
// if err != nil {
//     log.Fatalln("Failed to browse:", err.Error())
// }

// <-ctx.Done()
// }
