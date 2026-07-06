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
	
	viper.SetDefault("sharing.folders.autoArchive", true)
	viper.SetDefault("sharing.folders.archiveFormat", "zip") // or tar.gz
	viper.SetDefault("sharing.folders.autoExtractOnReceipt", true)

	viper.SetDefault("discovery.instanceName", hostname)
	viper.SetDefault("discovery.advanced.serviceName", "_drop._tcp")
	viper.SetDefault("discovery.advanced.domain", "local.")
	viper.SetDefault("discovery.advanced.metadata", []string{"txtv=1", "message = i made poopy in my pants"})

	viper.SetDefault("network.maxBandwidthMBps", 0) //0 is unlimited
}