package proxy

import (
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
)

func TestClient(t *testing.T) {
	c := NewHTTPClient("tcp", "localhost:7777", "tcp", "localhost:8081")
	rep, err := c.Get("http://localhost:8081/swagger")
	if err != nil {
		t.Fatal("Could not Get ", err)
	}
	t.Log("Status  ", rep.Status)
	//rep.Write(os.Stdout)

}

func TestWriteHeader(t *testing.T) {
	ErrorHTTP(200, "OK good", os.Stdout)
}
func executeNHTTPRequest(b *testing.B, c http.Client, req *http.Request) {
	for i := 0; i < b.N; i++ {
		rep, err := c.Do(req)
		if err != nil {
			b.Fatal("Could not Get ", err)
		}
		io.Copy(ioutil.Discard, rep.Body)
		rep.Body.Close()
	}
}

//To run benchmark test launch:
//
//go test  -bench ^BenchmarkTest -benchtime 30000x
func BenchmarkTestDirectHTTP(b *testing.B) {
	c := http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost:5566/test", nil)
	executeNHTTPRequest(b, c, req)
}

func BenchmarkTestHttpd(b *testing.B) {
	c := http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	executeNHTTPRequest(b, c, req)
}

func BenchmarkTestDirectUNIX(b *testing.B) {
	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/tmp/testhttp.sock")
			},
		},
	}
	req, _ := http.NewRequest("GET", "http://unix/test", nil)
	executeNHTTPRequest(b, c, req)
}

func BenchmarkTestProxyTcpTcp(b *testing.B) {
	c := NewHTTPClient("tcp", "localhost:7777", "tcp", "localhost:5566")
	req, _ := http.NewRequest("GET", "http://localhost:5566/test", nil)
	executeNHTTPRequest(b, c, req)
}

func BenchmarkTestProxyTcpUnix(b *testing.B) {
	c := NewHTTPClient("tcp", "localhost:7777", "unix", "/tmp/testhttp.sock")
	req, _ := http.NewRequest("GET", "http://localhost:5566/test", nil)
	executeNHTTPRequest(b, c, req)
}
