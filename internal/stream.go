package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
)

type ProgressWriter struct {
	TotalBytes int64
	Err        error
	Program   *tea.Program
	FileSize   int64
	LastSent  time.Time
}

type customReader struct {
    r   io.Reader
    buf []byte
}

func (cr *customReader) Read(p []byte) (int, error) {
    return cr.r.Read(p)
}

func (cr *customReader) WriteTo(w io.Writer) (int64, error) {
    return io.CopyBuffer(w, cr.r, cr.buf)
}

// i was initially sending a message to the progress model everytime a chunk of data was read
// but i think that was too much overhead, so i'm updating the progress model every 100ms instead
// i have no idea if this has an actual impact on the speed since it's not really a syscall
// but fuck it we ball
func (progWriter *ProgressWriter) Write(p []byte) (n int, err error) {
	progWriter.TotalBytes += int64(len(p))
	now := time.Now()
    if now.Sub(progWriter.LastSent) >= 100*time.Millisecond || progWriter.TotalBytes == progWriter.FileSize {
        progWriter.Program.Send(progressMsg{Decimal: float64(progWriter.TotalBytes) / float64(progWriter.FileSize)})
        progWriter.LastSent = now
    }
	return len(p), nil
}

func StreamFile(deviceAddress string, deviceName string, filePath string, program *tea.Program) error {
    file, err := os.Open(filePath)
    if err != nil {
        program.Send(doneMsg{Err: err})
        return fmt.Errorf("%s Error opening file: %v", Icons.Negative, err)
    }
    defer file.Close()
    
    fileInfo, err := os.Stat(filePath)
    if err != nil {
        program.Send(doneMsg{Err: err})
        return fmt.Errorf("%s Error opening file \"%s\": %v\n", Icons.Negative, filePath, err)
    }
    
    reader := io.TeeReader(file, &ProgressWriter{TotalBytes: 0, FileSize: fileInfo.Size(), Program: program})
    bodyReader := &customReader{r: reader, buf: make([]byte, 1024*1024)}
	// this is a 1MB buffer, will probably have a user-facing option to change this in the future.
	
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:3000/upload", deviceAddress), bodyReader)
    if err != nil {
        program.Send(doneMsg{Err: err})
        return fmt.Errorf("%s Error creating request: %v", Icons.Negative, err)
    }

    req.Header.Set("X-Filename", filepath.Base(filePath))
    req.Header.Set("X-Filesize", fmt.Sprintf("%d", fileInfo.Size()))
    req.ContentLength = fileInfo.Size()
    httpClient := &http.Client{}
    response, err := httpClient.Do(req)

    if err != nil {
        program.Send(doneMsg{Err: err})
        return fmt.Errorf("%s Error sending file: %v", Icons.Negative, err)
    }
    defer response.Body.Close()

    if response.StatusCode != http.StatusOK {
        fmt.Printf("%s Failed to send file. Status code: %d", Icons.Negative, response.StatusCode)
        program.Send(doneMsg{Err: fmt.Errorf("Failed to send file. Status code: %d", response.StatusCode)})
    } else if response.StatusCode == http.StatusOK {
        fmt.Printf("\r\033[2K%s The file \"%s\" has been sent successfully to %s.\n", Icons.Positive, filepath.Base(filePath), deviceName)
        
    program.Send(doneMsg{Err: err})
    }
    return nil
}