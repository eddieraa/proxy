package gnet

import (
	"github.com/panjf2000/gnet"
)

//Server proxy server instance
type Server struct {
	gnet.EventServer
	network string
	address string
	up      bool
}

//ListenAndServe listen and serve proxy
func (p *Server) ListenAndServe() (err error) {
	es := &Server{}

	gnet.Serve(es, p.address)
	return
}
