package server

import (
	"bufio"
	"log"
	"net/http"
	"net/url"
)

type FlushingWriter struct {
	Bufrw *bufio.ReadWriter
}

func (fw *FlushingWriter) Write(p []byte) (n int, err error) {
	n, err = fw.Bufrw.Write(p)
	if err != nil {
		return n, err
	}
	err = fw.Bufrw.Flush()
	return n, err
}

//mitm echoes the HTTP request/response to specified endpoint.
func Mitm(tu *url.URL) func(w http.ResponseWriter, r *http.Request) {
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
			// pres.Body.Close()?
			http.Error(w, "Host", http.StatusInternalServerError)
			return
		}

		hj, ok := w.(http.Hijacker)
		if !ok {
			log.Printf("webserver doesn't support hijacking\n")
			pres.Body.Close()
			http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			log.Printf("unable to hijack request %s\n", err)
			pres.Body.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Don't forget to close the connection:
		defer conn.Close()
		fw := new(FlushingWriter)
		fw.Bufrw = bufrw
		pres.Write(fw)
	}
}
