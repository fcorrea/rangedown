package rangedownload

import (
	"bytes"
	"errors"
	"io/ioutil"
	"math/rand"
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

// Client error returns an Error when Do is called
type ClientError struct{}

func (c *ClientError) Do(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: 500,
		Body:       ioutil.NopCloser(bytes.NewBufferString("BAD")),
		Header:     make(http.Header),
	}
	return resp, errors.New("Bad Request")
}

// ReaderError returns an Error when Read is called
type ReaderError struct{}

func (f *ReaderError) Read(p []byte) (n int, err error) {
	return 0, errors.New("Bad")
}

func TestNewRangeDownload(t *testing.T) {
	assert := assert.New(t)

	rangedownload := NewRangeDownload("http://foo.com/some.iso", http.DefaultClient)
	assert.Equal(rangedownload.URL, "http://foo.com/some.iso")
}

// Random strings for content generation. ref.: https://bit.ly/2OI5CfR
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func TestRangeDownloadStartBadURL(t *testing.T) {
	assert := assert.New(t)

	var result error
	rangedownload := NewRangeDownload("123%45%6", http.DefaultClient)
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

	rangedownload := NewRangeDownload("http://foo.com/some.iso", http.DefaultClient)
	rangedownload.client = client
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	go rangedownload.Start(out, errchn)
}

func TestRangeDownloadStartFailedRequest(t *testing.T) {
	assert := assert.New(t)

	var result error
	client := &ClientError{}
	rangedownload := NewRangeDownload("http://foo.com/some.iso", client)
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	done := make(chan bool)

	go func() {
		result = <-errchn
		done <- true
	}()

	go rangedownload.Start(out, errchn)

	<-done

	assert.Equal("Could not perform a request to http://foo.com/some.iso", result.Error())
}

func TestRangeDownloadStartBadContentLength(t *testing.T) {
	assert := assert.New(t)

	client := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal("http://foo.com/some.iso", req.URL.String())
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString("OK")),
			Header:     make(http.Header),
		}
		resp.Header.Set("Content-Length", "%%**")
		return resp
	})

	var result error
	rangedownload := NewRangeDownload("http://foo.com/some.iso", client)
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	done := make(chan bool)

	go func() {
		result = <-errchn
		done <- true
	}()

	go rangedownload.Start(out, errchn)

	<-done

	assert.Equal("Could not parse: %%**", result.Error())
}

func TestRangeDownloadStartCorruptDownload(t *testing.T) {
	assert := assert.New(t)

	client := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal("http://foo.com/some.iso", req.URL.String())
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString("OK")),
			Header:     make(http.Header),
		}
		resp.Header.Set("Content-Length", "100")
		return resp
	})

	var result error
	rangedownload := NewRangeDownload("http://foo.com/some.iso", client)
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	done := make(chan bool)

	go func() {
		result = <-errchn
		done <- true
	}()

	go rangedownload.Start(out, errchn)

	<-done

	assert.Equal("Corrupt download", result.Error())
}

func TestRangeDownloadStartBadResponseBody(t *testing.T) {
	assert := assert.New(t)

	client := NewTestClient(func(req *http.Request) *http.Response {
		re := &ReaderError{}
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(re),
			Header:     make(http.Header),
		}
		resp.Header.Set("Content-Length", "100")
		return resp
	})

	var result error
	rangedownload := NewRangeDownload("http://foo.com/some.iso", client)
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	done := make(chan bool)

	go func() {
		result = <-errchn
		done <- true
	}()

	go rangedownload.Start(out, errchn)

	<-done

	assert.Equal("Failed reading response body", result.Error())
}

func TestRangeDownloadStartReadsAllContent(t *testing.T) {
	assert := assert.New(t)

	content := RandStringBytes(20)
	client := NewTestClient(func(req *http.Request) *http.Response {
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(content)),
			Header:     make(http.Header),
		}
		resp.Header.Set("Content-Length", string(len(content)))
		return resp
	})

	var result []byte
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	done := make(chan bool)
	rangedownload := NewRangeDownload("http://foo.com/some.iso", client)

	go func() {
		result = <-out
		done <- true
	}()

	go rangedownload.Start(out, errchn)

	<-done

	assert.Equal(content, string(result))
}
