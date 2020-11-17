package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/eddieraa/proxy"
	opts "github.com/eddieraa/proxy/registry"
	"github.com/eddieraa/registry"
	pb "github.com/eddieraa/registry/nats"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

func main() {
	var natsURL, bindAddress, name string
	var debug bool

	flag.StringVar(&natsURL, "nats-url", "localhost:4222", "Nats URL, usage: proxy --nats-url=10.10.33.2:4222")
	flag.StringVar(&bindAddress, "address", "0.0.0.0:3128", "tcp bind address, usage: proxy --address=0.0.0.0:8585")
	flag.StringVar(&name, "name", "proxy", "service name, the proxy will register it self with this name")
	flag.BoolVar(&debug, "debug", false, "Activate debug informations")

	flag.Parse()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	conn, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal("Unable to connect to nats server: ", err)
	}
	options := []registry.Option{
		pb.Nats(conn),
		registry.AddFilter(registry.LoadBalanceFilter()),
		registry.SetObserverEvent(getObserveEvent()),
		registry.AddObserveFilter(registry.LocalhostOFilter()),
	}
	reg, err := registry.Connect(options...)
	if err != nil {
		log.Fatal("Unable to create registry client: ", err)
	}
	reg.Observe("*")

	proxyService := registry.Service{
		Address: bindAddress,
		Name:    name,
		Network: "tcp",
	}

	unregister, err := reg.Register(proxyService)
	if err != nil {
		panic(err)
	}
	defer unregister()

	proxyServer := proxy.NewServer(proxyService.Network, proxyService.Address, opts.NewRegistryOption(reg))
	err = proxyServer.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

//ObserveEvent func called when service change state
func getObserveEvent() registry.ObserverEvent {
	hostname, err := os.Hostname()
	if err != nil {
		logrus.Error("Could not get hostname, ", err)
		return nil
	}
	return func(s registry.Service, ev registry.Event) {
		logrus.Debugf("receive observe %s event for service %s on address %s", ev, s.Name, s.Address)
		if strings.HasPrefix(s.Address, "proxy:") {
			return
		}
		if s.Name == "proxy" {
			return
		}
		r, _ := registry.Connect()
		s.Host = hostname
		s.Network = "tcp"
		port := "-1"
		if offset := strings.Index(s.Address, ":"); offset != -1 {
			port = s.Address[offset+1:]
		}
		s.Address = "proxy:" + port
		if ev == registry.Register {
			r.Register(s)
		} else if ev == registry.Unregister {
			r.Unregister(s)
		}
	}
}
