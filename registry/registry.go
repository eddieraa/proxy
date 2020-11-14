package registry

import (
	"net/http"

	"github.com/eddieraa/proxy"
	"github.com/eddieraa/registry"
)

//NewRegistryFunc return FctService based on registry
func NewRegistryFunc(registry registry.Registry) proxy.FctService {
	findService := func(action string, args ...string) (*proxy.Service, error) {
		if action != "service" {
			return nil, proxy.ErrUnknownProtocol
		}
		if args == nil || len(args) == 0 || args[0] == "" {
			return nil, proxy.ErrInvalidArg
		}
		s, err := registry.GetService(args[0])
		if err != nil {
			return nil, err
		}
		network := s.Network
		if network == "" {
			network = "tcp"
		}
		return &proxy.Service{Network: network, Address: s.Address}, nil
	}

	return findService
}

//NewRegistryOption return registry option
func NewRegistryOption(registry registry.Registry) proxy.Option {
	return proxy.AddServiceOption(NewRegistryFunc(registry))
}

func NewHttpClient(service string) (cli http.Client, err error) {
	var reg registry.Registry
	if reg, err = registry.Connect(); err != nil {
		return
	}
	var s *registry.Service
	if s, err = reg.GetService("proxy"); err != nil {
		return
	}
	return proxy.NewHTTPClient(s.Network, s.Address, "service", service), nil

}
