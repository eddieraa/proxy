package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eddieraa/registry"
	rnats "github.com/eddieraa/registry/nats"

	"github.com/nats-io/nats.go"

	log "github.com/sirupsen/logrus"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

type s struct{}

func (s *s) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler(w, r)
}
func main() {
	sockaddr := flag.String("unix-addr", "/tmp/testshttp.sock", "unix socket file")
	tcpaddr := flag.String("tcp-addr", ":5566", "ip:port to listen")
	flag.Parse()
	http.HandleFunc("/test", handler)
	log.SetLevel(log.DebugLevel)

	s := &http.Server{
		Handler: &s{},
	}

	unixListener, err := net.Listen("unix", *sockaddr)
	if err != nil {
		panic(err)
	}
	defer func() {
		unixListener.Close()
		log.Info("Close unixListener ", unixListener)
	}()

	c, err := nats.Connect("localhost:4222")
	if err != nil {
		log.Fatal("Could not connect to nats: ", err)
	}

	reg, err := registry.SetDefault(rnats.Nats(c), registry.RegisterInterval(time.Second*1))
	if err != nil {
		log.Fatal("Could not open registry session", err)
	}
	host, _ := os.Hostname()
	go func() {
		unregister, err := reg.Register(registry.Service{Name: "httptest", Address: *sockaddr, Network: "unix", Host: host})
		if err != nil {
			log.Fatal("Could not register the service ", err)
		}
		defer unregister()

		if err = s.Serve(unixListener); err != nil {
			log.Fatal("Could not start server: ", err)
		}
	}()
	SetupCloseHandler(s, reg)
	reg.Register(registry.Service{Name: "httptest", Address: *tcpaddr, Network: "tcp", Host: host})
	if err = http.ListenAndServe(*tcpaddr, nil); err != nil {
		log.Fatal("Could not start tcp web server: ", err)
	}
	log.Info("END")
}

//SetupCloseHandler listen to ctrl-C
func SetupCloseHandler(s *http.Server, reg registry.Registry) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		reg.Close()
		s.Shutdown(context.Background())

		os.Exit(0)
	}()
}
