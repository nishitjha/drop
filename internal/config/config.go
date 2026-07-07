package config

import (
	"filepath"
	"os"

	"github.com/spf13/viper"
)

func init() {
	home, _ := os.UserHomeDir()
	hostname, _ := os.Hostname()

	viper.AddConfigPath(home)
	viper.SetConfigType("yaml")
	viper.SetConfigName(".drop")

	viper.SetDefault("webserver.port", 3000)
	viper.SetDefault("webserver.receiveDir", filepath.Join(home, "Downloads", "Drop"))

	viper.SetDefault("sharing.isDiscoverable", true)
	viper.SetDefault("sharing.askReceiveDirEverytime", false)
	viper.SetDefault("sharing.trustAllDevices", false)
	viper.SetDefault("sharing.trustedDevices", []string{})
	viper.SetDefault("sharing.rejectUntrustedDevices", false)
	viper.SetDefault("sharing.autoRenameExistingFiles", true)
	
	viper.SetDefault("sharing.acceptTextSnippets", true)
	viper.SetDefault("sharing.autoCopyToClipboard", true) 
	
	viper.SetDefault("sharing.advanced.enableTransferLog", true)
	viper.SetDefault("sharing.advanced.logFilePath", filepath.Join(home, ".drop_history.log"))
	
	viper.SetDefault("sharing.folders.archiveFormat", "zip") // or tar.gz or something else idk ask the user
	viper.SetDefault("sharing.folders.compressionLevel", 0) // 0 is no compression, 9 is max compression.
	// i personally think a lot of users are morons and must be informed that compression is a very CPU intensive task
	// otherwise everyone's gonna think "omg yeah i wanna compress it cause that way the transfer will be faster" no you fucking clown
	// at high internet speeds you should not compress at all because the bandwidth is not the bottleneck
	// at low internet speeds i think compression is worth it so as to reduce the size being trasnferred over
	// but idt any compression should be done by default so 0 it is 
	viper.SetDefault("sharing.folders.autoExtractOnReceiving", true)

	viper.SetDefault("discovery.instanceName", hostname)
	viper.SetDefault("discovery.advanced.serviceName", "_drop._tcp")
	viper.SetDefault("discovery.advanced.domain", "local.")
	viper.SetDefault("discovery.advanced.metadata", []string{"txtv=1", "message = i made poopy in my pants"})

	viper.SetDefault("network.maxBandwidthMBps", 0) //0 is unlimited
}