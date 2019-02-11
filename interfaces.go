package rangedown

import (
	"net/http"
	"os"
)

// Wrap the client to make it easier to test
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// FileOpener has the same method signature of os.OpenFile and helps with unit testing
type FileOpener func(string, int, os.FileMode) (*os.File, error)
