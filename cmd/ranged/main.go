package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/fcorrea/rangedown"
)

var url string

func init() {
	const (
		URLHelper = "The URL"
	)
	flag.StringVar(&url, "url", "", URLHelper)
	flag.StringVar(&url, "u", "", URLHelper)
}

func checkArgs(a []string) error {
	if len(a) == 0 {
		return errors.New("")
	}
	return nil
}

// Download creates and starts a new download
func Download(url string) (string, error) {
	// Create the download
	download, err := rangedown.NewDownload(url)
	if err != nil {
		return "", err
	}

	// Start downloading it
	download.Start()

	// Wait and check for progress
	written, err := download.Wait()
	if err != nil {
		panic(err.Error())
	}
	return fmt.Sprintf("Downloaded %v successfully. %d bytes written.", download.FileName, written), nil
}

func main() {
	flag.Parse()

	err := checkArgs(flag.Args())
	if err != nil {
		flag.Usage()
		os.Exit(1)
	}

	result, err := Download(url)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(result)
}
