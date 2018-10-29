package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/fcorrea/rangedownload"
)

var debug = flag.Bool("debug", false, "verbose output")

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatal("Please, provide the download URL as the first argument.")
	}

	download := rangedownload.New(flag.Arg(0))
	resp, err := download.GetSize()
	if err != nil {
		log.Fatal("Something went pretty bad.")
	}
	fmt.Println(resp)

}
