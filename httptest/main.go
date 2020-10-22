package main

import (
	"fmt"
	"net"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

type s struct{}

func (s *s) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler(w, r)
}
func main() {
	http.HandleFunc("/test", handler)

	s := &http.Server{
		Handler: &s{},
	}
	unixListener, err := net.Listen("unix", "/tmp/testhttp.sock")
	if err != nil {
		panic(err)
	}
	go func() {
		s.Serve(unixListener)
	}()

	http.ListenAndServe("127.0.0.1:5566", nil)
}
