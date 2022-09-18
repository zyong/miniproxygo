package main

import (
	"flag"
	"github.com/zyong/miniproxygo/m_config"
	"github.com/zyong/miniproxygo/m_server"
	"os"
	"os/signal"
)

func main() {

	flag.Parse()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	conf, _ := m_config.NewConfig(*confpath)
	server := m_server.NewServer()

	server.Bind(conf.Serv.Addr)
	if conf.Serv.Goroutinepool {
		server.WithGoroutinePool()
	}
	if conf.Serv.Cpunum {
		server.WithNumCPU()
	}
	server.Start()
	select {}
}
