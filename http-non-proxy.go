//http-non-proxy listens for http and makes an https request
// rewriting Host: header
package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
)

var f = flag.String("f", "", "From. server listen address. E.g. localhost:8000.")
var t = flag.String("t", "", "To. The server to route requests to. E.g. https://www.google.com")
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
	http.HandleFunc("/", handler(tu))
	log.Fatal(http.ListenAndServe(*f, nil))
}

//handler echoes the HTTP request/response to specified endpoint.
func handler(tu *url.URL) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		//log.Printf("Recieved %v request for %v", r.Method, r.RequestURI)
		defer r.Body.Close()
		preq, err := http.NewRequest(r.Method, tu.Scheme+"://"+tu.Host+r.RequestURI, r.Body)
		if err != nil {
			log.Printf("Unable to create request %s\n", err)
			http.Error(w, "Host", http.StatusInternalServerError)
			return
		}
		preq.Header = r.Header
		preq.Host = tu.Host
		pres, err := http.DefaultClient.Do(preq)
		if err != nil {
			log.Printf("Unable to make request %s\n", err)
			http.Error(w, "Host", http.StatusInternalServerError)
			return
		}

		hj, ok := w.(http.Hijacker)
		if !ok {
			log.Printf("webserver doesn't support hijacking\n")
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			log.Printf("Unable to hijack request %s\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Don't forget to close the connection:
		defer conn.Close()
		pres.Write(bufrw)
		bufrw.Flush()
	}
}
