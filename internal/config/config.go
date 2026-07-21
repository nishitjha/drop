package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func Launch() error {
	home, _ := os.UserHomeDir()
	hostname, _ := os.Hostname()

	viper.AddConfigPath(home)
	viper.SetConfigType("yaml")
	viper.SetConfigName(".drop")
	configFile := filepath.Join(home, ".drop.yaml")
	viper.SetConfigFile(configFile)

	viper.SetDefault("webserver.port", 3000)

	// default receive directory is ~/Downloads/Drop
	// but i'm wondering if one should have the option to have the file saved to the current working directory
	viper.SetDefault("sharing.receiveDir", filepath.Join(home, "Downloads", "Drop"))
	viper.SetDefault("sharing.isDiscoverable", true)
	viper.SetDefault("sharing.askReceiveDirEverytime", false)
	viper.SetDefault("sharing.trustAllDevices", false)
	viper.SetDefault("sharing.askToTrustEverytime", true)
	viper.SetDefault("sharing.trustedDevices", []string{})
	viper.SetDefault("sharing.autoRejectUntrustedDevices", false)
	viper.SetDefault("sharing.autoRenameExistingFiles", true)

	viper.SetDefault("sharing.acceptTextSnippetsByDefault", false)
	viper.SetDefault("sharing.autoCopyToClipboard", true)

	viper.SetDefault("sharing.advanced.enableTransferLog", true)
	viper.SetDefault("sharing.advanced.logFilePath", filepath.Join(home, ".drop_history.log"))

	viper.SetDefault("sharing.folders.archiveFormat", func() string {
		switch runtime.GOOS {
		case "windows":
			return "zip"
		case "darwin", "linux":
			//return "tar.gz"
			return "zip" // just for now, will implement later i promise
		default:
			return "zip"
		}
	}())
	// using zip is better for cross-platform compatibility, but tar.gz is better for speed
	// using archive/tar also allows for write-as-you-receive streaming on the webserver endpoint while zip does not
	// so it is worth mentioning that if the user is not on windows, they should probably use tar.gz instead of zip

	viper.SetDefault("sharing.folders.compressionLevel", 0) // 0 is no compression, 1 is best speed w minimal compression and so on till 3
	// most users are fucking morons and will probably think "omg yeah i wanna compress my files so that they take lesser time to stream across"
	// but compression is very CPU intensive and will actually slow it down for most users
	// imo for users with a fast internet connection, compression is a waste of CPU cycles and the bandwidth is not the bottleneck
	// if you have a slow internet connection, compression will probably help
	// either way the default should be 0 ngl

	viper.SetDefault("sharing.folders.intelligentArchive", false)
	// it's not all that intelligent lolol but will def lead to increased speeds
	// turning it on will skip mp4, mkv, avi, jpg and png files etc which would not benefit greatly from compression
	// will also exclude .zip, .tar.gz, .rar, .7z and other archive formats
	// from what i read online compression (if turned on at all) is helpful only for .txt, .json, .xml, .csv and other text-based files
	// it will also EXCLUDE directories like node_modules, .git, .svn, .hg, .vscode, .idea etc to save time
	viper.SetDefault("sharing.folders.autoExtractOnReceive", true)

	viper.SetDefault("discovery.instanceName", hostname)
	viper.SetDefault("discovery.advanced.serviceName", "_drop._tcp")
	viper.SetDefault("discovery.advanced.domain", "local.")
	viper.SetDefault("discovery.advanced.port", 3001)
	viper.SetDefault("discovery.advanced.deviceUUID", uuid.New().String())
	viper.SetDefault("network.maxBandwidthMBps", 0) //0 is unlimited

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, os.ErrNotExist) {
			if err := viper.SafeWriteConfig(); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
