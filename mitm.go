//mitm listens for http and makes an https request
// rewriting Host: header
package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/libliflin/mitm/server"
)

var f = flag.String("f", "", "From. The server listen address. E.g. localhost:8000.")
var t = flag.String("t", "", "To.   The server to route requests to. E.g. https://www.google.com")
var tu *url.URL

func main() {
	flag.Parse()
	if *f == "" {
		log.Print("From required.")
		flag.Usage()
		os.Exit(1)
	}
	if *t == "" {
		log.Print("To required.")
		flag.Usage()
		os.Exit(1)
	}
	tu, err := url.Parse(*t)
	if err != nil {
		log.Fatalf("To address given (%v) is not a valid url %v", *t, err)
	}
	if tu.Scheme != "https" && tu.Scheme != "http" {
		log.Fatalf("To address given (%v) does not have http or https as scheme", *t)
	}
	http.HandleFunc("/", server.Mitm(tu))
	log.Fatal(http.ListenAndServe(*f, nil))
}
