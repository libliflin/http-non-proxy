package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"

	"github.com/libliflin/mitm/server"
)

func main() {
	os.Exit(mtimtest())
}

type SynchronizedUpload struct {
	firstUpload      []byte
	firstUploadSent  int
	lockBetween      sync.Mutex
	secondUpload     []byte
	secondUploadSent int
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (su *SynchronizedUpload) Read(p []byte) (n int, err error) {
	canSendFirst := len(su.firstUpload) - su.firstUploadSent
	canSendSecond := len(su.secondUpload) - su.secondUploadSent
	// can send First
	if canSendFirst > 0 {
		sending := min(len(p), canSendFirst)
		copy(p, su.firstUpload[su.firstUploadSent:su.firstUploadSent+sending])
		su.firstUploadSent += sending
		return sending, nil
	}
	// done with first, have we started second?
	if su.secondUploadSent == 0 {
		log.Printf("UPLOAD:   waiting on client after first file upload chunk sent until it is verified on server.\n")
		su.lockBetween.Lock()
		log.Printf("UPLOAD:   lock unlocked on client, sending second chunk.\n")
	}
	if canSendSecond > 0 {
		sending := min(len(p), canSendSecond)
		copy(p, su.secondUpload[su.secondUploadSent:su.secondUploadSent+sending])
		su.secondUploadSent += sending
		return sending, nil
	}
	return 0, io.EOF
}

func mtimtest() int {
	log.Println("Testing mitm streaming with locks.")
	// serveMu: lock the mutex while serving content
	var serveMu sync.Mutex
	givenGreetingBase := []byte("Hello, client")
	serveMult := 10
	givenGreetingLen := len(givenGreetingBase) * serveMult
	givenGreeting := make([]byte, 0, givenGreetingLen)
	for i := 0; i < serveMult; i++ {
		givenGreeting = append(givenGreeting, givenGreetingBase...)
	}
	firstGreetingLen := len(givenGreeting) / 2
	firstGreeting := givenGreeting[:firstGreetingLen]
	secondGreeting := givenGreeting[firstGreetingLen:]
	serveMu.Lock()
	log.Printf("DOWNLOAD: lock initialized as locked\n")

	// uploadMu: lock the mutex while uploading content
	su := new(SynchronizedUpload)
	su.lockBetween.Lock()
	log.Printf("UPLOAD:   lock initialized as locked\n")
	uploadBase := []byte("tolstoy")
	uploadMult := 10
	uploadLen := len(uploadBase) * uploadMult
	upload := make([]byte, 0, uploadLen)
	for i := 0; i < uploadMult; i++ {
		upload = append(upload, uploadBase...)
	}
	firstUploadLen := len(upload) / 2
	su.firstUpload = upload[:firstUploadLen]
	su.secondUpload = upload[firstUploadLen:]

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		resFirstUpload := make([]byte, len(su.firstUpload), len(su.firstUpload))
		readIntoResult := 0
		for readIntoResult < len(su.firstUpload) {
			n, err := r.Body.Read(resFirstUpload[readIntoResult:])
			readIntoResult += n
			if err != nil {
				log.Fatalf("unable to read first upload %v\n", err)
			}
		}
		if fmt.Sprintf("%q", resFirstUpload) != fmt.Sprintf("%q", su.firstUpload) {
			log.Fatalf("mitm server received %q (%v bytes) first while it was expected to receive %q", resFirstUpload, readIntoResult, su.firstUpload)
		}
		log.Printf("UPLOAD:   first upload chunk verified, unlocking upload lock.\n")
		su.lockBetween.Unlock()
		resSecondUpload := make([]byte, len(su.firstUpload), len(su.secondUpload))
		readIntoResult = 0
		for readIntoResult < len(su.secondUpload) {
			n, err := r.Body.Read(resSecondUpload[readIntoResult:])
			readIntoResult += n
			if err != nil {
				log.Fatalf("unable to read second upload %v\n", err)
			}
		}
		if fmt.Sprintf("%q", resSecondUpload) != fmt.Sprintf("%q", su.secondUpload) {
			log.Fatalf("mitm server received %q second while it was expected to receive %q", resSecondUpload, su.secondUpload)
		}
		log.Printf("UPLOAD:   upload stream check completed.")

		n, err := w.Write(firstGreeting)
		if err != nil {
			log.Fatalf("unable to send greeting\n")
		}
		if n != len(firstGreeting) {
			log.Fatalf("expected to send %v but sent %v bytes.\n", len(firstGreeting), n)
		}
		fl, ok := w.(http.Flusher)
		if !ok {
			log.Fatalf("webserver doesn't support flushing\n")
		}
		fl.Flush()
		log.Printf("DOWNLOAD: locking on server after flushing first greeting.\n")
		serveMu.Lock()
		log.Printf("DOWNLOAD: lock unlocked on server, sending second greeting.\n")
		fmt.Fprint(w, string(secondGreeting))
		serveMu.Unlock()
	}))
	defer ts.Close()

	tsurl, err := url.Parse(ts.URL)
	if err != nil {
		log.Fatalf("unable to parse local test url %v\n", ts.URL)
	}

	mitmts := httptest.NewServer(http.HandlerFunc(server.Mitm(tsurl)))
	defer mitmts.Close()

	req, err := http.NewRequest("POST", mitmts.URL, su)
	if err != nil {
		log.Fatalf("unable to create request to server %v\n", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("unable to get mitm server %v\n", err)
	}

	resFirstGreeting := make([]byte, len(firstGreeting), len(firstGreeting))
	_, err = res.Body.Read(resFirstGreeting)
	if err != nil {
		log.Fatalf("unable to read first Greeting %v\n", err)
	}
	if fmt.Sprintf("%q", resFirstGreeting) != fmt.Sprintf("%q", firstGreeting) {
		log.Fatalf("mitm server served %q first while it was expected to server %q", resFirstGreeting, firstGreeting)
	}
	log.Printf("DOWNLOAD: unlocking on client after validating first greeting.\n")
	serveMu.Unlock()
	resSecondGreeting := make([]byte, len(firstGreeting), len(secondGreeting))
	_, err = res.Body.Read(resSecondGreeting)
	if err != nil && err != io.EOF {
		log.Fatalf("unable to read second Greeting %v", err)
	}
	if fmt.Sprintf("%q", resSecondGreeting) != fmt.Sprintf("%q", secondGreeting) {
		log.Fatalf("mitm server served %q second while it was expected to server %q", resSecondGreeting, secondGreeting)
	}
	log.Printf("DOWNLOAD: verification of second greeting complete.\n")

	log.Println("All tests passed.")
	return 0
}
