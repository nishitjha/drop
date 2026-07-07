package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func Launch() error {
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

	viper.SetDefault("sharing.folders.archiveFormat", "zip") // or tar.gz
	viper.SetDefault("sharing.folders.compressionLevel", 0)  // 0 is no compression, 1 is best speed w minimal compression and so on
	// most users are fucking morons and will probably think "omg yeah i wanna compress my files so that they take lesser time to stream across"
	// but compression is very CPU intensive and will actually slow it down for most users
	// imo for users with a fast internet connection, compression is a waste of CPU cycles and the bandwidth is not the bottleneck
	// if you have a slow internet connection, compression will probably help
	// either way the default should be 0 ngl
	viper.SetDefault("sharing.folders.autoExtractOnReceive", true)

	viper.SetDefault("discovery.instanceName", hostname)
	viper.SetDefault("discovery.advanced.serviceName", "_drop._tcp")
	viper.SetDefault("discovery.advanced.domain", "local.")
	viper.SetDefault("discovery.advanced.metadata", []string{"txtv=1", "message = i made poopy in my pants"})
	viper.SetDefault("discovery.advanced.port", 3001)
	viper.SetDefault("network.maxBandwidthMBps", 0) //0 is unlimited

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		// no config file yet
		}	
		return nil
	}
	