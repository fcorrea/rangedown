package rangedown

import (
	"bytes"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
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

func NewTestableChunk(dURL string, client HttpClient) *Chunk {
	u, _ := url.Parse(dURL)
	download, _ := NewChunk(u)
	download.client = client
	download.opener = OpenTempFile
	download.outChn = make(chan []byte, 1)
	download.errChn = make(chan error, 1)
	return download
}

func TestChunkStartCorrectURL(t *testing.T) {
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

	download := NewTestableChunk("http://foo.com/some.iso", client)

	download.Start()
}

func TestChunkStartFailedRequest(t *testing.T) {
	assert := assert.New(t)

	client := &ClientError{}
	download := NewTestableChunk("http://foo.com/some.iso", client)

	var result error
	done := make(chan bool)

	go func() {
		result = <-download.errChn
		done <- true
	}()

	download.Start()

	<-done

	assert.Equal("Could not perform a request to http://foo.com/some.iso", result.Error())
}

func TestChunkStartCorruptChunk(t *testing.T) {
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

	download := NewTestableChunk("http://foo.com/some.iso", client)

	var result error
	done := make(chan bool)

	go func() {
		result = <-download.errChn
		done <- true
	}()

	download.Start()

	<-done

	assert.Equal("Corrupt download", result.Error())
}

func TestChunkStartBadResponseBody(t *testing.T) {
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

	download := NewTestableChunk("http://foo.com/some.iso", client)

	var result error
	done := make(chan bool)

	go func() {
		result = <-download.errChn
		done <- true
	}()

	download.Start()

	<-done

	assert.Equal("Failed reading response body", result.Error())
}

func TestChunkStartReadsAllContent(t *testing.T) {
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

	download := NewTestableChunk("http://foo.com/some.iso", client)

	var result []byte
	done := make(chan bool)

	download.Start()

	go func() {
		for v := range download.outChn {
			result = append(result, v...)
		}
		done <- true
	}()

	<-done

	assert.Equal(content, string(result))
}

func TestChunkWait(t *testing.T) {
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

	download := NewTestableChunk("http://foo.com/some.iso", client)

	download.Start()

	err := download.Wait()
	if err != nil {
		panic(err)
	}

	result, _ := ioutil.ReadFile(download.File.Name())

	assert.Equal("some.iso", download.FileName)
	assert.Equal(content, string(result))
	defer os.Remove(download.File.Name())
}

func TestChunkWaitOpenFileError(t *testing.T) {
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

	download := NewTestableChunk("http://foo.com/some.iso", client)
	download.opener = FileOpenerWithError

	download.Start()

	err := download.Wait()
	assert.Equal("A file error", err.Error())
}

func TestChunkWriteError(t *testing.T) {
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

	download := NewTestableChunk("http://foo.com/some.iso", client)
	download.opener = FileOpenerWithWriteError

	download.Start()

	err := download.Wait()
	assert.NotNil(err)
}

func TestChunkChunks(t *testing.T) {
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

	download := NewTestableChunk("http://foo.com/some.iso", client)
	download.opener = FileOpenerWithWriteError

	download.Start()

	err := download.Wait()
	assert.NotNil(err)
}

func TestNewDownload(t *testing.T) {
	assert := assert.New(t)

	download, _ := NewDownload("http://foo.com/some.iso", 16)
	assert.Equal(download.URL.Scheme, "http")
	assert.Equal(download.URL.Host, "foo.com")
	assert.Equal(download.URL.Path, "/some.iso")
	assert.Equal(download.ParallelConnections, 16)
}

func TestNewDownloadBadURL(t *testing.T) {
	assert := assert.New(t)

	_, err := NewDownload("123%45%6", 16)
	assert.NotNil(err)
	assert.Equal("parse 123%45%6: invalid URL escape \"%6\"", err.Error())

}
