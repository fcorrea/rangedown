package rangedownload

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Wrap the client to make it easier to test
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Rangedownload holds information about a download
type RangeDownload struct {
	URL    string
	client HttpClient
}

// NewRangeDownload initializes a RangeDownload with the url
func NewRangeDownload(url string, client HttpClient) *RangeDownload {
	return &RangeDownload{
		URL:    url,
		client: client,
	}
}

// Start starts downloading the requested URL and sending the read bytes into
// the out channel
func (r *RangeDownload) Start(out chan<- []byte, errchn chan<- error) {
	defer close(out)
	defer close(errchn)
	var read int64
	// Build the request
	url, err := url.Parse(r.URL)
	if err != nil {
		errchn <- fmt.Errorf("Could not parse URL: %v", r.URL)
	}
	req := &http.Request{
		URL:    url,
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
