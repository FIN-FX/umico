package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	RemoteAddrReqLimit  = 20
	RemoteAddrReqPeriod = 10 * time.Second
)

type Counters struct {
	sync.RWMutex
	remoteAddr        map[string]uint64
	remoteAddrNextTry map[string]time.Time
}

var counters = &Counters{
	remoteAddr:        make(map[string]uint64),
	remoteAddrNextTry: make(map[string]time.Time),
}

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer counters.Unlock()
		counters.Lock()
		if counters.remoteAddrNextTry[req.RemoteAddr].Sub(time.Now()) > 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if counters.remoteAddr[req.RemoteAddr] >= RemoteAddrReqLimit {
			counters.remoteAddr[req.RemoteAddr] = 0
			counters.remoteAddrNextTry[req.RemoteAddr] = time.Now().Add(RemoteAddrReqPeriod)
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		counters.remoteAddr[req.RemoteAddr]++
		next.ServeHTTP(w, req)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", RateLimitMiddleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("OK"))
		}),
	))
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalln(err)
	}
}
