package rangy

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// Wrap the client to make it easier to test
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Rangy holds information about a download
type RangyDownload struct {
	URL         *url.URL
	client      HttpClient
	writeCloser io.WriteCloser
	FileName    string
}

// NewRangyDownload initializes a RangyDownload with downloadURL
func NewRangyDownload(downloadURL string, client HttpClient) *RangyDownload {
	p, err := url.Parse(downloadURL)
	if err != nil {
		panic("Could not parse URL: " + downloadURL)
	}

	return &RangyDownload{
		URL:      p,
		client:   client,
		FileName: filepath.Base(p.Path),
	}
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
	for d := range data {
		dw, err := r.writeCloser.Write(d)
		if err != nil {
			return 0, err
		}
		written += int64(dw)
	}
	defer r.writeCloser.Close()
	return written, nil
}

// SetupWriter creates a file using the file name stored in RangyDownload
func (r *RangyDownload) SetupWriter() error {
	f, err := os.Create(r.FileName)
	if err != nil {
		return errors.New("Could not create " + r.FileName)
	}
	r.writeCloser = f
	return nil
}
