package archive

import (
	"archive/zip"
	"io"
	"net/http"
	"path/filepath"
	"runtime"

	//"archive/tar"
	"compress/flate"
	"fmt"
	"os"

	"github.com/nishitjha/drop/internal"
	"github.com/spf13/viper"
)

func getCompressionLevel(level int) int {
	switch level {
	case 0:
		return flate.NoCompression
	case 1:
		return flate.BestSpeed
	case 2:
		return flate.DefaultCompression
	case 3:
		return flate.BestCompression
	default:
		return flate.DefaultCompression
	}
}

func Execute(sourceDir string, targetAddress string, targetDeviceName string) {
	archiveFormat := viper.GetString("sharing.folders.archiveFormat")

	if archiveFormat == "zip" && (runtime.GOOS == "linux" || runtime.GOOS == "darwin") {
		fmt.Printf("%s Using the zip archive format in a Unix environment. Consider switching to tar.gz for greatly improved speeds. Use \"drop config sharing.folders.archiveFormat\" to learn more.\n", internal.Icons.Warning)
	}

	pr, pw := io.Pipe()

	go func () {
		err := ArchiveDirectoryToZip(sourceDir, pw)
		
		if err != nil {
			fmt.Printf("%s Error archiving directory \"%s\": %v.\n", internal.Icons.Negative, sourceDir, err)
			pw.CloseWithError(err)
			return 
		}

		pw.Close()
	}()

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:3000/archive?format=%s", targetAddress, archiveFormat), pr)
	if err != nil {
		fmt.Printf("%s Error creating request to send directory \"%s\" to device \"%s\": %v.\n", internal.Icons.Negative, sourceDir, targetDeviceName, err)
		return
	}

	req.Header.Set("X-Filename", filepath.Base(sourceDir)+"_drop.zip")
	req.Header.Set("Content-Type", "application/zip")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s Error sending request to device \"%s\": %v.\n", internal.Icons.Negative, targetDeviceName, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("%s Failed to send directory \"%s\" to device \"%s\". Status code: %d.\n", internal.Icons.Negative, sourceDir, targetDeviceName, resp.StatusCode)
		return
	} else {
		fmt.Printf("%s The directory \"%s\" has been sent successfully to \"%s\".\n", internal.Icons.Positive, sourceDir, targetDeviceName)
		return
	}
} 

func ArchiveDirectoryToZip(sourceDir string, destination io.Writer) error {
	// archive, err := os.Create(fmt.Sprintf("%s_drop.zip", sourceDir))
	// if err != nil {  
	// 	return err
	
	// }
	// defer archive.Close()

	// READ ->
	// i think using io.Pipe() is better than making a temp archive
	// mainly because it allows me to stream the archive directly instead of making a gazillion syscalls
	// but the issue is that I cannot have any sort of progress indicators if I use io.Pipe()
	// that's because I cannot possibly know the size of the archive if I stream it concurrently
	// it is definitely a tradeoff between speed and user experience
	// two ideas I can come up with rn:
	// 1. have a user configurable option that asks whether they value realtime progress indication or a speed boost (only significant for big folders tho)   
	// 2. use some hacky estimation algorithm to figure out the size of the archive before streaming it based on the level of compression and the types of files in the folder

	zipWriter := zip.NewWriter(destination)
	defer zipWriter.Close()

	configLevel := viper.GetInt("sharing.folders.compressionLevel")
	compLevel := getCompressionLevel(configLevel)

	fmt.Printf("Using compression level: %d\n", compLevel)

	zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, compLevel)
	})
	
	intelligentArchive := viper.GetBool("sharing.folders.intelligentArchive")
	fmt.Printf("%[1]s %[2]s intelligent archiving. Use \"drop config sharing.folders.intelligentArchive\" to learn more.\n", func() string {
		if intelligentArchive {
			return internal.Icons.Positive
		}
		return internal.Icons.Warning
	}(), func() string {
		if intelligentArchive {
			return "Using"
		}
		return "Not using"
	}())

	if intelligentArchive {
		IntelligentArchive(sourceDir, zipWriter)
	} else {
		dirFS := os.DirFS(sourceDir)
		err := zipWriter.AddFS(dirFS) 

		if err != nil {
			return err
		}
	}
	
	return nil
}

func IntelligentArchive(sourceDir string, zipWriter *zip.Writer) error {
	err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

