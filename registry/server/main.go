package main

import (
	"github.com/eddieraa/proxy"
	opts "github.com/eddieraa/proxy/registry"
	"github.com/eddieraa/registry"
	"github.com/nats-io/nats.go"
)

func main() {
	conn, err := nats.Connect("localhost:4222")
	if err != nil {
		panic(err)
	}
	reg, err := registry.Connect(registry.Nats(conn))
	if err != nil {
		panic(err)
	}

	proxyService := registry.Service{
		Address: "0.0.0.0:8383",
		Name:    "proxy",
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
