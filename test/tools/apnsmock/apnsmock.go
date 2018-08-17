package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/kayac/Gunfish/mock"
)

func main() {
	var (
		port              int
		keyFile, certFile string
		verbose           bool
	)

	flag.IntVar(&port, "port", 2195, "apns mock server port")
	flag.StringVar(&keyFile, "cert-file", "", "apns mock server key file")
	flag.StringVar(&certFile, "key-file", "", "apns mock server cert file")
	flag.BoolVar(&verbose, "verbose", false, "verbose flag")
	flag.Parse()

	mux := mock.APNsMockServer(verbose)
	log.Println("start apnsmock server")
	if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", port), keyFile, certFile, mux); err != nil {
		log.Fatal(err)
	}
}
