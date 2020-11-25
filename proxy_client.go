package proxy

import (
	"fmt"
	"net"
	"net/http"
)

//Client create this struct for create new HttpClient
type Client struct {
	//proxy address
	proxyAddress string
	//tcp|unix
	proxyNetwork string
	//network tcp|unix|service
	serviceNetwork string
	//service address or service name
	service string
}

//NewHTTPClient init proxy client with transport and return new http client
//
//proxyNetwork: tcp|unix
//
//service: address or service name
func NewHTTPClient(proxyNetwork, proxyAddress string, serviceNetwork, service string) http.Client {
	proxyClient := Client{proxyNetwork: proxyNetwork, proxyAddress: proxyAddress, serviceNetwork: serviceNetwork, service: service}
	trans := &http.Transport{
		Dial:              proxyClient.proxyDial,
		DisableKeepAlives: false,
	}
	return http.Client{Transport: trans}
}

//proxyDial connect to proxy server
//send to proxy network and destination address to the proxy
//
//tcp 127.0.0.1:5676\r
//
//unix /tmp/myservice.sock\r
func (p Client) proxyDial(network, address string) (net.Conn, error) {
	conn, err := net.Dial(p.proxyNetwork, p.proxyAddress)
	if err == nil {
		_, err = conn.Write([]byte(fmt.Sprintf("%s %s\r", p.serviceNetwork, p.service)))

	}
	return conn, err
}
