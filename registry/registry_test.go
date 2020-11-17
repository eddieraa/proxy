package registry

import (
	"io"
	"os"
	"testing"

	"github.com/eddieraa/registry"
	pb "github.com/eddieraa/registry/nats"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

var conn *nats.Conn

func initRegistry() {
	if conn != nil {
		return
	}
	conn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	registry.Connect(pb.Nats(conn))
}
func TestClientWithRegistry(t *testing.T) {
	initRegistry()
	request(t)
	request(t)
	request(t)
}

func request(t *testing.T) {
	c, err := NewHttpClient("httptest")
	if err != nil {
		t.Fatal(err)
	}

	rep, err := c.Get("http://popo/lolo/toto.html")
	if err != nil {
		t.Fatal("Could not Get ", err)
	}
	logrus.Info("Status  ", rep.Status)
	io.Copy(os.Stdout, rep.Body)
}
