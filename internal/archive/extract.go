package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func ExtractArchive(archivePath string, destinationDir string, dirName string) error {

	// make sure the destinationDir exists (but MkdirAll will not overwrite it if it already exists)
	err := os.MkdirAll(destinationDir, os.ModePerm)
	if err != nil {
		return err
	}
	// create the staging directory
	stagingDir := filepath.Join(destinationDir,"_"+dirName +"_temp_" + uuid.New().String())
	err = os.MkdirAll(stagingDir, os.ModePerm)
	if err != nil {
		return err
	}

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	for _, file := range archive.File {
		filePath, err := safeJoin(stagingDir, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
        	err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
            	return err
        	}
        	continue
    	}

		err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		if err != nil {
        	return err
    	}

		dest, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode()) 
		// open the destination file with the same permissions as the original file
		// but this will rewrite the file if it already exists, which is not intended but i'll fix it later :zany:

		if err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			dest.Close()
			return err
		}

		_, err = io.Copy(dest, src)
		if err != nil {
			dest.Close()
			src.Close()
			return err
		}

		dest.Close()
		src.Close()
	}

    finalPath := filepath.Join(destinationDir, dirName)
	err = RenameStagingDir(stagingDir, finalPath)
	if err != nil {
		return err
	}
	
	return nil
}

func RenameStagingDir(stagingDir string, newName string) error {
	err := os.Rename(stagingDir, newName)
	if err != nil {
		return err
	}
	return nil
}

func safeJoin(basePath string, relativePath string) (string, error) {
	target := filepath.Join(basePath, filepath.Clean("/"+relativePath))

	// this check was claude's idea hats off to my goat
    if !strings.HasPrefix(target, filepath.Clean(basePath)+string(os.PathSeparator)) {
        return "", fmt.Errorf("illegal file path: %s", relativePath)
    }
    return target, nil
}