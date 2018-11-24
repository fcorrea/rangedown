package rangy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// Rangy holds information about a download
type RangyDownload struct {
	URL      *url.URL
	File     *os.File
	FileName string
	client   HttpClient
	opener   FileOpener
}

// Wrap the client to make it easier to test
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// FileOpener has the same signature of os.OpenFile and helps with unit testing
type FileOpener func(string, int, os.FileMode) (*os.File, error)

// NewRangyDownload initializes a RangyDownload with downloadURL and set up the download file.
func NewRangyDownload(downloadURL string) (*RangyDownload, error) {
	p, err := url.Parse(downloadURL)
	if err != nil {
		return nil, err
	}
	return &RangyDownload{
		URL:    p,
		client: http.DefaultClient,
		opener: os.OpenFile,
	}, nil
}

// Start starts downloading the requested URL and as soon as data start being read,
// it sends it in the out channel
func (r *RangyDownload) Start(out chan<- []byte, errchn chan<- error) {
	defer close(out)
	defer close(errchn)

	var read int64

	// Build the request
	req := &http.Request{
		URL:    r.URL,
		Method: "GET",
		Header: make(http.Header),
	}

	// Perform the request
	resp, err := r.client.Do(req)
	if err != nil {
		errchn <- fmt.Errorf("Could not perform a request to %v", r.URL)
	}
	defer resp.Body.Close()

	// Start consuming the response body
	size := resp.ContentLength
	for {
		buf := make([]byte, 4*1024)
		br, err := resp.Body.Read(buf)
		if br > 0 {
			// Increment the bytes read and send the buffer out to be written
			read += int64(br)
			out <- buf[0:br]
		}
		if err != nil {
			// Check for possible end of file indicating end of the download
			if err == io.EOF {
				if read != size {
					errchn <- fmt.Errorf("Corrupt download")
				}
				break
			} else {
				errchn <- fmt.Errorf("Failed reading response body")
			}
		}
	}
}

// Write will read from data channel and write it to the file
func (r *RangyDownload) Write(data <-chan []byte) (int64, error) {
	var written int64

	// Setup file for download
	fileName := filepath.Base(r.URL.Path)
	r.FileName = fileName
	f, err := r.opener(fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return 0, err
	}
	r.File = f

	for d := range data {
		dw, err := r.File.Write(d)
		if err != nil {
			return 0, err
		}
		written += int64(dw)
	}
	defer r.File.Close()
	return written, nil
}
