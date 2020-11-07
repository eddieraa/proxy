package proxy

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

//proxy proxy struct
type proxy struct {
	in, out net.Conn
}

type errinfo struct {
	err     error
	service string
}

var (
	//ErrInvalidArg Invalid argument
	ErrInvalidArg = errors.New("Invalid arguments")
	//ErrUnknownProtocol Protocol not supproted
	ErrUnknownProtocol = errors.New("Protocol not supported for this function")
)

//NewProxy return new proxy
func newProxy(in, out net.Conn) *proxy {
	p := &proxy{in: in, out: out}
	return p
}

func (p *proxy) copy(in io.ReadCloser, out io.WriteCloser) {
	_, err := io.Copy(out, in)
	if err != nil {
		logrus.Info(err)
	}
	defer in.Close()
	defer out.Close()
}

//Launch launch the proxy
func (p *proxy) Launch() {
	go p.copy(p.in, p.out)
	p.copy(p.out, p.in)

}

//Parse first bytes of stream until '\r' and return array of string, first trame look like this :
//
//
//SERVICE configd opt1=value1 opt2=value2\r....
//
//TCP localhost:5000 \r....
//
//UNIX /tmp/GEN.confid.sock\r....
//
//SHUTDOWN GRACEFULLY\r
//
func parse(in io.Reader) (string, string, []string, error) {
	var sb strings.Builder
	var err error
	res := []string{}
	var l int
	var c byte
	b := make([]byte, 1)
	t := ""
	arg1 := ""
	for err == nil {
		if l, err = in.Read(b); err == nil && l == 1 {
			c = b[0]
			if c == '\r' || c == ' ' {
				if t == "" {
					t = sb.String()
				} else if arg1 == "" {
					arg1 = sb.String()
				} else {
					res = append(res, sb.String())
				}
				sb.Reset()
			} else {
				sb.WriteByte(c)
			}
			if c == '\r' {
				break
			}

		}
	}

	return t, arg1, res, err
}

//Server proxy server instance
type Server struct {
	network     string
	address     string
	up          bool
	listener    net.Listener
	fctServices []FctService
	opts        Options
}

//Service service network (tcp, unix, udp, ....)
type Service struct {
	Network string
	Address string
}

//FctService return Service if supported
type FctService func(action string, args ...string) (service *Service, err error)

//NewServer return new Server
func NewServer(network, address string, opts ...Option) *Server {
	s := &Server{
		network:     network,
		address:     address,
		fctServices: []FctService{fctServiceTCP, fctServiceUNIX},
		opts:        newOptions(opts...),
	}
	s.fctServices = append(s.fctServices, s.opts.fcts...)

	return s
}

func fctServiceTCP(action string, args ...string) (*Service, error) {
	if action != "tcp" {
		return nil, ErrUnknownProtocol
	}
	if args == nil || len(args) == 0 || args[0] == "" {
		return nil, ErrInvalidArg
	}
	return &Service{Network: "tcp", Address: args[0]}, nil
}

func fctServiceUNIX(action string, args ...string) (*Service, error) {
	if action != "unix" {
		return nil, ErrUnknownProtocol
	}
	if args == nil || len(args) == 0 || args[0] == "" {
		return nil, ErrInvalidArg
	}
	return &Service{Network: "unix", Address: args[0]}, nil
}

//ListenAndServe listen and serve proxy
func (p *Server) ListenAndServe() (err error) {
	p.listener, err = net.Listen(p.network, p.address)
	if err != nil {
		return err
	}
	defer p.listener.Close()
	p.up = true
	logrus.Print("Start proxy ready on address ", p.address, " network ", p.network)

	for p.up {
		var conn net.Conn

		conn, err = p.listener.Accept()
		if err != nil {
			logrus.Errorf("Could not accept new incomming conn: %s", err.Error())
			continue
		}
		logrus.Print("Accept new incoming connection from ", conn.LocalAddr())
		go handleNewIncoming(conn, p.fctServices)

	}
	return nil
}

//Stop stop the proxy server
func (p *Server) Stop() {
	p.up = false
	if p.listener != nil {
		if err := p.listener.Close(); err != nil {
			logrus.Errorf("Could not stop listener %s", err.Error())
		}
	}

}

func handleNewIncoming(conn net.Conn, fcts []FctService) {
	action, arg1, _, err := parse(conn)
	logrus.Printf("Parse action %s, arg1 %s", action, arg1)
	if err != nil {
		logrus.Errorf("Could not parse new incomming header: %s", err.Error())
		return
	}

	var service *Service
	for _, f := range fcts {
		service, err = f(action, arg1)
		if err == ErrUnknownProtocol {
			continue
		}
		if err != nil {
			logrus.Errorf("Invalid incoming header no ARG for %s action", err)
			break
		}
		if service != nil {
			break
		}
	}
	if service == nil {
		logrus.Error("Could not parse header for unknown error")
		return
	}

	outconn, err := net.Dial(service.Network, service.Address)
	if err != nil {
		err = fmt.Errorf("Could not tcp dial to %s from (%s): %v", arg1, conn.RemoteAddr().String(), err)
		ErrorHTTP(503, err.Error(), conn)
		logrus.Error(err)
		return
	}
	newProxy(conn, outconn).Launch()

}

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

//ErrorHTTP write HTTP response
func ErrorHTTP(errCode int, errString string, out io.Writer) error {
	r := http.Response{
		Status:     errString,
		StatusCode: errCode,
		Header:     http.Header{},
	}
	t := time.Now()
	r.Header.Set("Date", t.Format(http.TimeFormat))
	r.Header.Set("Server", "Proxy")
	r.Write(out)
	return nil
}
