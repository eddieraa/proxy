package registry

import (
	"io"
	"io/ioutil"
	"testing"

	"github.com/eddieraa/registry"
	"github.com/nats-io/nats.go"
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
	registry.Connect(registry.Nats(conn))
}
func TestClientWithRegistry(t *testing.T) {
	initRegistry()
	c, err := NewHttpClient("httptest")
	if err != nil {
		t.Fatal(err)
	}

	rep, err := c.Get("http://localhost:8081/swagger")
	if err != nil {
		t.Fatal("Could not Get ", err)
	}
	t.Log("Status  ", rep.Status)
	io.Copy(ioutil.Discard, rep.Body)
}
