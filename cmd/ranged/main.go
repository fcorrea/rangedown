package main

import (
	"flag"
	"fmt"

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

func main() {
	flag.Parse()

	// Create the download
	download, err := rangedown.NewDownload(url)
	if err != nil {
		panic(err.Error)
	}

	// Start downloading it
	download.Start()

	// Wait and check for progress
	err = download.Wait()
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("Finished!")
}
