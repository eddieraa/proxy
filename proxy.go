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
	up      *bool
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
	//ErrStopRequested Stop action, sent by a client for stopping this server
	ErrStopRequested = errors.New("Stop is requested by the client")
)

//NewProxy return new proxy
func newProxy(in, out net.Conn, up *bool) *proxy {
	p := &proxy{in: in, out: out, up: up}
	return p
}

func (p *proxy) copy(src io.ReadCloser, dst io.WriteCloser, up *bool) {
	var written int64
	var err error
	buf := make([]byte, 32*1024)
	for up == nil || *up {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	if err != nil {
		logrus.Info(err)
	}
	src.Close()
	dst.Close()
}

//Launch launch the proxy
func (p *proxy) Launch() {
	go p.copy(p.in, p.out, nil)
	p.copy(p.out, p.in, p.up)

}

//Parse first bytes of stream until '\r' and return array of string, first trame look like this :
//
//
//service configd opt1=value1 opt2=value2\r....
//
//tcp localhost:5000 \r....
//
//unix /tmp/GEN.confid.sock\r....
//
//stop GRACEFULLY\r
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
		fctServices: []FctService{fctServiceTCP, fctServiceUNIX, fctServiceSTOP},
		opts:        newOptions(opts...),
	}
	s.fctServices = append(s.fctServices, s.opts.fcts...)

	return s
}

func fctBasicCtrl(action, expectedAction string, args ...string) error {
	if action != expectedAction {
		return ErrUnknownProtocol
	}
	if args == nil || len(args) == 0 || args[0] == "" {
		return ErrInvalidArg
	}
	return nil
}

func fctServiceTCP(action string, args ...string) (*Service, error) {
	if err := fctBasicCtrl(action, "tcp", args...); err != nil {
		return nil, err
	}
	return &Service{Network: "tcp", Address: args[0]}, nil
}

func fctServiceUNIX(action string, args ...string) (*Service, error) {
	if err := fctBasicCtrl(action, "unix", args...); err != nil {
		return nil, err
	}
	return &Service{Network: "unix", Address: args[0]}, nil
}

func fctServiceSTOP(action string, args ...string) (*Service, error) {
	if err := fctBasicCtrl(action, "stop", args...); err != nil {
		return nil, err
	}
	return nil, ErrStopRequested
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
		go handleNewIncoming(conn, p.fctServices, &p.up)

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

func handleNewIncoming(conn net.Conn, fcts []FctService, up *bool) {
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
			logrus.Errorf("Invalid incoming header action (%s) arg1 (%s), err: %s", action, arg1, err)
			break
		}
		if service != nil {
			break
		}
	}
	if service == nil {
		if err == nil {
			logrus.Error("Could not parse header for unknown error")
			err = ErrInvalidArg
		}
		ErrorHTTP(503, err.Error(), conn)

		return
	}

	outconn, err := net.Dial(service.Network, service.Address)
	if err != nil {
		err = fmt.Errorf("Could not tcp dial for service (%s) to (%s:%s) from (%s): %v", arg1, service.Network, service.Address, conn.RemoteAddr().String(), err)
		ErrorHTTP(503, err.Error(), conn)
		logrus.Error(err)
		return
	}
	newProxy(conn, outconn, up).Launch()

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
