package pullit

import (
	"errors"
	"net/http"
	"net/url"
)

type Pullit struct {
	URL    string
	url    *url.URL
	Client *http.Client
}

// New initializes a Pullit struct with the url
func New(url string) *Pullit {
	return &Pullit{
		URL: url,
	}
}

// CheckClient ensures the code will have a http client
func (p *Pullit) CheckClient() error {
	if p.Client == nil {
		p.Client = http.DefaultClient
	}

	return nil
}

// MakeRequest creates a return a http.Request
func (p *Pullit) MakeRequest() (*http.Request, error) {
	var err error
	if p.url == nil {
		p.url, err = url.Parse(p.URL)
		if err != nil {
			return nil, err
		}
	}
	return &http.Request{
		Method:     "GET",
		URL:        p.url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       nil,
		Host:       p.url.Host,
	}, nil
}

// GetSize makes a HEAD request and check for the file size
func (p *Pullit) GetSize() (int64, error) {
	if err := p.CheckClient(); err != nil {
		return 0, err
	}

	req, err := p.MakeRequest()
	if err != nil {
		return 0, err
	}
	req.Method = "HEAD"

	resp, err := p.Client.Do(req)
	if err != nil {
		return 0, err
	}

	if resp.ContentLength < 0 {
		return 0, errors.New("no content length for GetSize()")
	}
	return resp.ContentLength, nil
}
