package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"

	"github.com/zyong/miniproxygo"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

var confpath = flag.String("confpath", "config/config-dev.yml", "server config file path")

func main() {

	flag.Parse()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go signalHandler(c)

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil { //监控cpu
			log.Fatal("could not start CPU profile: ", err)
		}
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC()                                      // GC，获取最新的数据信息
		if err := pprof.WriteHeapProfile(f); err != nil { // 写入内存信息
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}

	conf, _ := miniproxygo.NewConfig(*confpath)
	server := miniproxygo.NewServer()

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

func signalHandler(c chan os.Signal) {
	<-c
	pprof.StopCPUProfile()
	os.Exit(0)
}
