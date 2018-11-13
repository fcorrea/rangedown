package rangedownload

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Replace Transport and make testing easier
type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func TestNewRangeDownload(t *testing.T) {
	assert := assert.New(t)

	rangedownload := NewRangeDownload("http://foo.com/some.iso")
	assert.Equal(rangedownload.URL, "http://foo.com/some.iso")
}

func TestRangeDownloadStartBadURL(t *testing.T) {
	assert := assert.New(t)

	var result error
	rangedownload := NewRangeDownload("123%45%6")
	out := make(chan []byte, 1)
	errchn := make(chan error)
	done := make(chan bool)

	go func() {
		result = <-errchn
		done <- true
	}()

	go rangedownload.Start(out, errchn)

	<-done

	assert.Equal("Could not parse URL: 123%45%6", result.Error())

}

func TestRangeDownloadStartCorrectURL(t *testing.T) {
	assert := assert.New(t)

	client := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal("http://foo.com/some.iso", req.URL.String())
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString("OK")),
			Header:     make(http.Header),
		}
		resp.Header.Set("Content-Length", "2")
		return resp
	})

	rangedownload := NewRangeDownload("http://foo.com/some.iso")
	rangedownload.client = client
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	go rangedownload.Start(out, errchn)
}
