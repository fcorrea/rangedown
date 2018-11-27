package main

import (
	"flag"
	"fmt"

	"github.com/fcorrea/rangedown"
)

func main() {
	var url string
	flag.StringVar(&url, "url", "", "The URL")

	flag.Parse()

	// Create the download
	download, err := rangedown.NewDownload(url)
	if err != nil {
		panic(err.Error())
	}

	// Start downloading it
	download.Start()

	// Wait and check for progress
	download.Wait()
	fmt.Println("Download complete.")
}
