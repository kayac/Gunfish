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
		port      int
		projectID string
		verbose   bool
	)

	flag.IntVar(&port, "port", 8888, "fcmv1 mock server port")
	flag.StringVar(&projectID, "project-id", "test", "fcmv1 mock project id")
	flag.BoolVar(&verbose, "verbose", false, "verbose flag")
	flag.Parse()

	mux := mock.FCMv1MockServer(projectID, verbose)
	log.Println("start fcmv1mock server port:", port, "project_id:", projectID)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		log.Fatal(err)
	}
}
