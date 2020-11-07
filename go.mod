module github.com/eddieraa/proxy

go 1.15

require (
	github.com/eddieraa/registry v0.0.2
	github.com/nats-io/nats.go v1.10.0
	github.com/sirupsen/logrus v1.7.0
)

replace github.com/eddieraa/registry => ../registry
