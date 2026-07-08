package archive

import (
	"archive/zip"
	"io"
	"path/filepath"

	//"archive/tar"
	"compress/flate"
	"fmt"
	"os"

	"github.com/nishitjha/drop/internal"
	"github.com/spf13/viper"
)

type Level struct {
	
}

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

func ArchiveDirectoryToZip(sourceDir string) error {
	archive, err := os.Create(fmt.Sprintf("%s_drop.zip", sourceDir))
	if err != nil {  
		return err
	
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	configLevel := viper.GetInt("sharing.folders.compressionLevel")
	compLevel := getCompressionLevel(configLevel)

	fmt.Printf("Using compression level: %d\n", compLevel)

	zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, compLevel)
	})
	
	intelligentArchive := viper.GetBool("sharing.folders.intelligentArchive")
	fmt.Printf("%[1]s %[2]s intelligent compression. Use \"drop config sharing.folders.intelligentArchive\" to learn more.\n", func() string {
		if intelligentArchive {
			return internal.Icons.Positive
		}
		return internal.Icons.Negative
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
		err = zipWriter.AddFS(dirFS) 

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