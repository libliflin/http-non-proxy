//mitm listens for http and makes an https request
// rewriting Host: header
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
)

var f = flag.String("f", "", "From. The server listen address. E.g. localhost:8000.")
var t = flag.String("t", "", "To.   The server to route requests to. E.g. https://www.google.com")
var test = flag.Bool("test", false, "Verify the server streams requests.")
var tu *url.URL

func main() {
	flag.Parse()
	if *test {
		os.Exit(mtimtest())
	}
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
	http.HandleFunc("/", mitm(tu))
	log.Fatal(http.ListenAndServe(*f, nil))
}

//mitm echoes the HTTP request/response to specified endpoint.
func mitm(tu *url.URL) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		//log.Printf("Recieved %v request for %v", r.Method, r.RequestURI)
		defer r.Body.Close()
		preq, err := http.NewRequest(r.Method, tu.Scheme+"://"+tu.Host+r.RequestURI, r.Body)
		if err != nil {
			log.Printf("unable to create request %s\n", err)
			http.Error(w, "Host", http.StatusInternalServerError)
			return
		}
		preq.Header = r.Header
		preq.Host = tu.Host
		pres, err := http.DefaultClient.Do(preq)
		if err != nil {
			log.Printf("unable to make request %s\n", err)
			http.Error(w, "Host", http.StatusInternalServerError)
			return
		}
		//DEBUG
		fmt.Println("made request to host.")

		hj, ok := w.(http.Hijacker)
		if !ok {
			log.Printf("webserver doesn't support hijacking\n")
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			log.Printf("unable to hijack request %s\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Don't forget to close the connection:
		defer conn.Close()
		pres.Write(bufrw)
		bufrw.Flush()
	}
}

func mtimtest() int {
	log.Println("Testing mitm")
	var mu sync.Mutex
	givenGreetingBase := []byte("Hello, client")
	mult := 1024 * 1024
	givenGreeting := make([]byte, len(givenGreetingBase)*mult, len(givenGreetingBase)*mult)
	for i := 0; i < mult; i++ {
		givenGreeting = append(givenGreeting, givenGreetingBase...)
	}
	firstGreeting := givenGreeting[:mult]
	secondGreeting := givenGreeting[mult:]
	mu.Lock()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(firstGreeting))
		fl, ok := w.(http.Flusher)
		if !ok {
			log.Fatalf("webserver doesn't support flushing\n")
		}
		fl.Flush()
		mu.Lock()
		fmt.Fprint(w, string(secondGreeting))
		mu.Unlock()
	}))
	defer ts.Close()

	tsurl, err := url.Parse(ts.URL)
	if err != nil {
		log.Fatalf("unable to parse local test url %v\n", ts.URL)
	}

	mitmts := httptest.NewServer(http.HandlerFunc(mitm(tsurl)))
	defer mitmts.Close()

	res, err := http.Get(mitmts.URL)
	if err != nil {
		log.Fatalf("unable to get mitm server %v\n", err)
	}
	resFirstGreeting := make([]byte, len(firstGreeting), len(firstGreeting))
	_, err = res.Body.Read(resFirstGreeting)
	if err != nil {
		log.Fatalf("unable to read first Greeting %v", err)
	}
	if fmt.Sprintf("%q", resFirstGreeting) != fmt.Sprintf("%q", firstGreeting) {
		log.Fatalf("mitm server served %q first while it was expected to server %q", resFirstGreeting, firstGreeting)
	}
	mu.Unlock()
	resSecondGreeting := make([]byte, len(secondGreeting), len(secondGreeting))
	_, err = res.Body.Read(resSecondGreeting)
	if err != nil {
		log.Fatalf("unable to read second Greeting %v", err)
	}
	if fmt.Sprintf("%q", resSecondGreeting) != fmt.Sprintf("%q", secondGreeting) {
		log.Fatalf("mitm server served %q first while it was expected to server %q", resSecondGreeting, secondGreeting)
	}

	log.Println("All tests passed.")
	return 0
}
