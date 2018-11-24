package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fcorrea/rangedown"
	"github.com/urfave/cli"
)

// StartDownload setups and starts downloading a Rangy
func StartDownload(url string) {
	out := make(chan []byte)
	errChan := make(chan error)
	download, err := rangedown.NewDownload(url)
	if err != nil {
		panic(err.Error())
	}

	go download.Start(out, errChan)
	fmt.Println("Download started")

	// If something bad happens while downloading, it will panic
	go func() {
		for v := range errChan {
			panic(v.Error())
		}
	}()

	written, err := download.Write(out)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Download of file %v complete. %d bytes written\r\n", download.FileName, written)
}

func main() {
	var url string
	app := cli.NewApp()
	app.Name = "ranged"
	app.Usage = "A lightweight, multi-connection file downloader written in Go"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "url, u",
			Usage:       "The URL of the file to be dowloaded",
			Destination: &url,
		},
	}

	app.Action = func(c *cli.Context) error {
		if url == "" {
			fmt.Println(cli.ShowAppHelp(c))
			os.Exit(1)
		}
		StartDownload(url)
		return nil

	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
