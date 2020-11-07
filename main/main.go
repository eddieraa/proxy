package main

import (
	"github.com/eddieraa/proxy"
	"github.com/sirupsen/logrus"
)

func main() {

	logrus.SetLevel(logrus.InfoLevel)

	go func() {
		pux := proxy.NewServer("unix", "/tmp/proxy.sock")
		if err := pux.ListenAndServe(); err != nil {
			logrus.Fatal(err)
		}
	}()

	p := proxy.NewServer("tcp", "localhost:7777")
	if err := p.ListenAndServe(); err != nil {
		logrus.Fatal(err)
	}
}
