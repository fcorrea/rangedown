package rangedown

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	tm "github.com/buger/goterm"
)

// Download holds information about a download
type Download struct {
	URL       *url.URL
	File      *os.File
	FileName  string
	client    HttpClient
	opener    FileOpener
	TotalSize int64
	outChn    chan []byte
	errChn    chan error
}

// Wrap the client to make it easier to test
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// FileOpener has the same method signature of os.OpenFile and helps with unit testing
type FileOpener func(string, int, os.FileMode) (*os.File, error)

// NewDownload initializes a Download with downloadURL, a default http client and an FileOpener
func NewDownload(downloadURL string) (*Download, error) {
	p, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}
	return &Download{
		URL:    p,
		client: http.DefaultClient,
		opener: os.OpenFile,
		outChn: make(chan []byte),
		errChn: make(chan error),
	}, nil
}

// Start starts downloading the requested URL and as soon as data start being read,
// it sends it out in the outChn channel
func (r *Download) Start() {
	var read int64

	// Build the request
	req := &http.Request{
		URL:    r.URL,
		Method: "GET",
		Header: make(http.Header),
	}

	go func() {
		defer close(r.outChn)
		defer close(r.errChn)

		// Perform the request
		resp, err := r.client.Do(req)
		if err != nil {
			r.errChn <- fmt.Errorf("Could not perform a request to %v", r.URL)
		}
		defer resp.Body.Close()

		// Start consuming the response body
		r.TotalSize = resp.ContentLength
		for {
			buf := make([]byte, 4*1024)
			br, err := resp.Body.Read(buf)
			if br > 0 {
				// Increment the bytes read and send the buffer out to be written
				read += int64(br)
				r.outChn <- buf[0:br]
			}
			if err != nil {
				// Check for possible end of file indicating end of the download
				if err == io.EOF {
					if read != r.TotalSize {
						r.errChn <- fmt.Errorf("Corrupt download")
					}
					break
				} else {
					r.errChn <- fmt.Errorf("Failed reading response body")
				}
			}
		}
	}()
}

// Wait reads on the outChan and writes it to the disk
func (r *Download) Wait(progress bool) error {
	var written int64

	// Setup file for download
	fileName := filepath.Base(r.URL.Path)
	r.FileName = fileName
	f, err := r.opener(fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	r.File = f

	for d := range r.outChn {
		dw, err := r.File.Write(d)
		if err != nil {
			return err
		}
		if progress == true {
			written += int64(dw)
			mib := int64(written / 1024 / 1024)
			perc := written * 100 / r.TotalSize

			tm.Clear()
			tm.MoveCursor(1, 1)
			tm.Printf("Downloading %v %v%% (%v/%v), %v MiB\n", r.FileName, perc, written, r.TotalSize, mib)
			tm.Flush()
			time.Sleep(5 * time.Nanosecond)
		}
	}
	defer r.File.Close()

	return nil
}
