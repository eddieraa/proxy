package registry

import (
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

		return &proxy.Service{Network: s.Network, Address: s.Address}, nil
	}

	return findService
}

//NewRegistryOption return registry option
func NewRegistryOption(registry registry.Registry) proxy.Option {
	return proxy.AddServiceOption(NewRegistryFunc(registry))
}
