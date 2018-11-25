package rangedown

import (
	"bytes"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

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

// Random strings for content generation. ref.: https://bit.ly/2OI5CfR
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// OpenTempFile is a FileOpener that creates a temp file instead of a local file
func OpenTempFile(name string, flags int, perm os.FileMode) (*os.File, error) {
	f, err := ioutil.TempFile("", name)
	if err != nil {
		panic(err.Error())
	}
	return f, nil
}

// FileOpenerWithError emulates a FileOpener that retuns an error
func FileOpenerWithError(name string, flags int, perm os.FileMode) (*os.File, error) {
	return nil, errors.New("A file error")
}

// FileOpenerWithWriteError emulates a FileOpener that returns a closed temp file
// so any calls to write will return an error
func FileOpenerWithWriteError(name string, flags int, perm os.FileMode) (*os.File, error) {
	f, _ := ioutil.TempFile("", name)
	defer f.Close()
	return f, nil
}

func NewTestableDownload(url string, client HttpClient) *Download {
	download, _ := NewDownload(url)
	download.client = client
	download.opener = OpenTempFile
	return download
}

func TestNewDownload(t *testing.T) {
	assert := assert.New(t)

	download, _ := NewDownload("http://foo.com/some.iso")
	assert.Equal(download.URL.Scheme, "http")
	assert.Equal(download.URL.Host, "foo.com")
	assert.Equal(download.URL.Path, "/some.iso")
}

func TestNewDownloadBadURL(t *testing.T) {
	assert := assert.New(t)

	_, err := NewDownload("123%45%6")
	assert.NotNil(err)
	assert.Equal("parse 123%45%6: invalid URL escape \"%6\"", err.Error())

}

func TestDownloadStartCorrectURL(t *testing.T) {
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

	download := NewTestableDownload("http://foo.com/some.iso", client)

	out := make(chan []byte, 1)
	errchn := make(chan error, 1)

	go download.Start(out, errchn)
}

func TestDownloadStartFailedRequest(t *testing.T) {
	assert := assert.New(t)

	client := &ClientError{}
	download := NewTestableDownload("http://foo.com/some.iso", client)

	var result error
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	done := make(chan bool)

	go func() {
		result = <-errchn
		done <- true
	}()

	go download.Start(out, errchn)

	<-done

	assert.Equal("Could not perform a request to http://foo.com/some.iso", result.Error())
}

func TestDownloadStartCorruptDownload(t *testing.T) {
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

	download := NewTestableDownload("http://foo.com/some.iso", client)

	var result error
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	done := make(chan bool)

	go func() {
		result = <-errchn
		done <- true
	}()

	go download.Start(out, errchn)

	<-done

	assert.Equal("Corrupt download", result.Error())
}

func TestDownloadStartBadResponseBody(t *testing.T) {
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

	download := NewTestableDownload("http://foo.com/some.iso", client)

	var result error
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	done := make(chan bool)

	go func() {
		result = <-errchn
		done <- true
	}()

	go download.Start(out, errchn)

	<-done

	assert.Equal("Failed reading response body", result.Error())
}

func TestDownloadStartReadsAllContent(t *testing.T) {
	assert := assert.New(t)

	rand.Seed(time.Now().UTC().UnixNano())
	content := RandStringBytes(5 * int(rand.Int31n(1000)))
	client := NewTestClient(func(req *http.Request) *http.Response {
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(content)),
			Header:     make(http.Header),
		}
		resp.ContentLength = int64(len(content))
		return resp
	})

	download := NewTestableDownload("http://foo.com/some.iso", client)

	var result []byte
	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	done := make(chan bool)

	go download.Start(out, errchn)

	go func() {
		for v := range out {
			result = append(result, v...)
		}
		done <- true
	}()

	<-done

	assert.Equal(content, string(result))
}

func TestDownloadWrite(t *testing.T) {
	assert := assert.New(t)

	rand.Seed(time.Now().UTC().UnixNano())
	content := RandStringBytes(5 * int(rand.Int31n(1000)))
	client := NewTestClient(func(req *http.Request) *http.Response {
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(content)),
			Header:     make(http.Header),
		}
		resp.ContentLength = int64(len(content))
		return resp
	})

	download := NewTestableDownload("http://foo.com/some.iso", client)

	out := make(chan []byte, 1)
	errchn := make(chan error, 1)
	go download.Start(out, errchn)

	written, err := download.Write(out)
	if err != nil {
		panic("could not write file " + err.Error())
	}

	result, err := ioutil.ReadFile(download.File.Name())
	if err != nil {
		panic("could not read file " + err.Error())
	}

	assert.Equal("some.iso", download.FileName)
	assert.Equal(int64(len(content)), written)
	assert.Equal(content, string(result))
	defer os.Remove(download.File.Name())
}

func TestDownloadWriteOpenFileError(t *testing.T) {
	assert := assert.New(t)

	content := RandStringBytes(10)
	client := NewTestClient(func(req *http.Request) *http.Response {
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(content)),
			Header:     make(http.Header),
		}
		resp.ContentLength = int64(len(content))
		return resp
	})

	download := NewTestableDownload("http://foo.com/some.iso", client)
	download.opener = FileOpenerWithError

	out := make(chan []byte, 1)
	errchn := make(chan error, 1)

	go download.Start(out, errchn)

	written, err := download.Write(out)
	assert.Equal(int64(0), written)
	assert.Equal("A file error", err.Error())
}

func TestDownloadWriteError(t *testing.T) {
	assert := assert.New(t)

	content := RandStringBytes(10)
	client := NewTestClient(func(req *http.Request) *http.Response {
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(content)),
			Header:     make(http.Header),
		}
		resp.ContentLength = int64(len(content))
		return resp
	})

	download := NewTestableDownload("http://foo.com/some.iso", client)
	download.opener = FileOpenerWithWriteError

	out := make(chan []byte, 1)
	errchn := make(chan error, 1)

	go download.Start(out, errchn)

	written, err := download.Write(out)
	assert.Equal(int64(0), written)
	assert.NotNil(err)
}
