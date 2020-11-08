module github.com/eddieraa/proxy

go 1.15

require (
	github.com/eddieraa/registry v0.0.5
	github.com/nats-io/nats.go v1.10.0
	github.com/panjf2000/gnet v1.3.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
)

replace github.com/eddieraa/registry => ../registry
