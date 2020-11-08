package main

import (
	"flag"
	"log"

	"github.com/eddieraa/proxy"
	opts "github.com/eddieraa/proxy/registry"
	"github.com/eddieraa/registry"
	"github.com/nats-io/nats.go"
)

func main() {
	var natsURL, bindAddress, name string
	flag.StringVar(&natsURL, "nats-url", "localhost:4222", "Nats URL, usage: proxy --nats-url=10.10.33.2:4222")
	flag.StringVar(&bindAddress, "address", "0.0.0.0:3128", "tcp bind address, usage: proxy --address=0.0.0.0:8585")
	flag.StringVar(&name, "name", "proxy", "service name, the proxy will register it self with this name")

	conn, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal("Unable to connect to nats server: ", err)
	}
	reg, err := registry.Connect(registry.Nats(conn))
	if err != nil {
		log.Fatal("Unable to create registry client: ", err)
	}

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
