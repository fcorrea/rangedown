package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/fcorrea/pullit"
)

var debug = flag.Bool("debug", false, "verbose output")

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatal("Please, provide the download URL as the first argument.")
	}

	p := pullit.New(flag.Arg(0))
	resp, err := p.GetSize()
	if err != nil {
		log.Fatal("Something went pretty bad.")
	}
	fmt.Println(resp)

}
