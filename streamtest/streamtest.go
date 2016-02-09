package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"time"

	"github.com/libliflin/mitm/server"
)

const (
	NUM_CHUNKS = 4
	STRINGSEED = "<(''<)(>''<)(>'')>"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	log.Println("Testing mitm streaming with locks.")
	serverChunks := RandomChunks("DOWNLOAD")
	uploadChunks := RandomChunks("UPLOAD")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request received")
		uploadChunks.readChunks(r.Body)
		log.Printf("%s upload stream check completed.", uploadChunks.mkr)
		serverChunks.sendChunks(w)
	}))
	defer ts.Close()

	tsurl, err := url.Parse(ts.URL)
	if err != nil {
		log.Fatalf("unable to parse local test url %v\n", ts.URL)
	}

	mitmts := httptest.NewServer(http.HandlerFunc(server.Mitm(tsurl)))
	defer mitmts.Close()

	req, err := http.NewRequest("POST", mitmts.URL, uploadChunks)
	if err != nil {
		log.Fatalf("unable to create request to server %v\n", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("unable to get mitm server %v\n", err)
	}

	serverChunks.readChunks(res.Body)
	log.Printf("DOWNLOAD: verification of second greeting complete.\n")

	log.Println("All tests passed.")
}

type Chunk struct {
	chunk []byte
	c     int
}

type Chunks struct {
	mu     sync.Mutex
	chunks []Chunk
	mkr    string
	i      int
}

// make n chunks starting with seed.
func RandomChunks(mkr string) *Chunks {
	cks := new(Chunks)
	cks.mkr = fmt.Sprintf("%-10v", mkr)
	cks.mu.Lock()
	log.Printf("%s lock initialized as locked\n", cks.mkr)
	cks.chunks = make([]Chunk, NUM_CHUNKS, NUM_CHUNKS)
	s := []byte(STRINGSEED)
	for i := 0; i < NUM_CHUNKS; i++ {
		rot7(s)
		log.Printf("%s s: %q", cks.mkr, s)
		var ck Chunk
		ck.chunk = make([]byte, 0, len(s))
		ck.chunk = append(ck.chunk, s...)
		cks.chunks[i] = ck
	}
	return cks
}

func (rc *Chunks) readChunks(reader io.Reader) {
	for i := 0; i < len(rc.chunks); i++ {
		log.Printf("%s verifying chunk [%d]", rc.mkr, i)
		ck := rc.chunks[i].chunk
		resChunk := make([]byte, len(ck), len(ck))
		readIntoResult := 0
		for readIntoResult < len(ck) {
			log.Printf("%s [%d] reading %d bytes, starting at %d\n", rc.mkr, i, len(ck), readIntoResult)
			n, err := reader.Read(resChunk[readIntoResult:])
			readIntoResult += n
			if err != nil && (readIntoResult != len(ck) || err != io.EOF) {
				log.Fatalf("%s unable to read [%d] %v\n", rc.mkr, rc.i, err)
			}
		}

		if fmt.Sprintf("%q", resChunk) != fmt.Sprintf("%q", ck) {
			log.Fatalf("%s received %q (%v bytes) [%d] while it was expected to receive %q", rc.mkr, resChunk, readIntoResult, rc.i, ck)
		}
		log.Printf("%s [%d] chunk verified, unlocking lock.\n", rc.mkr, i)
		rc.mu.Unlock()
	}
}

func (rc *Chunks) Read(p []byte) (n int, err error) {
	for i := 0; i < len(rc.chunks); i++ {
		chunk := rc.chunks[i].chunk
		c := rc.chunks[i].c
		log.Printf("%s [%d] chunk: %q", rc.mkr, i, chunk)
		log.Printf("%s [%d] c: %d", rc.mkr, i, c)
		left := len(chunk) - c
		if i > 0 && c == 0 {
			log.Printf("%s waiting on client after file upload chunk [%d] sent until it is verified on server.\n", rc.mkr, i-1)
			rc.mu.Lock()
			log.Printf("%s lock unlocked on client, sending [%d] chunk.\n", rc.mkr, i)
		}
		if left > 0 {
			sending := min(len(p), left)
			copy(p, chunk[c:c+sending])
			c += sending
			log.Printf("%s sending %d bytes", rc.mkr, sending)
			rc.chunks[i].c = c
			return sending, nil
		}
	}
	return 0, io.EOF
}

func (rc *Chunks) sendChunks(w io.Writer) {
	fl, ok := w.(http.Flusher)
	if !ok {
		log.Fatalf("webserver doesn't support flushing\n")
	}
	for i := 0; i < len(rc.chunks); i++ {
		ck := rc.chunks[i].chunk
		if i > 0 {
			log.Printf("%s locking on server after flushing [%d] greeting.\n", rc.mkr, i-1)
			rc.mu.Lock()
			log.Printf("%s lock unlocked on server, sending [%d] greeting.\n", rc.mkr, i)
		}
		n, err := w.Write(ck)
		if err != nil {
			log.Fatalf("unable to send greeting\n")
		}
		if n != len(ck) {
			log.Fatalf("expected to send %v but sent %v bytes.\n", len(ck), n)
		}
		fl.Flush()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func rot7(p []byte) {
	for i := 0; i < len(p); i++ {
		// reset all chars to 'a' if not letters.
		if (p[i] < 'A' || 'Z' < p[i]) && (p[i] < 'a' || 'z' < p[i]) {
			p[i] = 'a' + byte(uint8(rand.Intn(26)))
		}
		p[i] += 7
		// wrap when out of range.
		if (p[i] < 'A' || 'Z' < p[i]) && (p[i] < 'a' || 'z' < p[i]) {
			p[i] -= 26
		}
	}
}
