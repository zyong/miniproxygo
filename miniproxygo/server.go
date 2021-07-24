package main

import (
	"flag"
	"fmt"
	"log"
	"miniproxygo/codec/http"
	"miniproxygo/handler"

	"github.com/panjf2000/gnet"
)

func main() {
	var port int
	var multicore bool

	// Example command: go run http.go --port 8080 --multicore=true
	flag.IntVar(&port, "port", 8080, "server port")
	flag.BoolVar(&multicore, "multicore", true, "multicore")
	flag.Parse()

	httpHandler := new(handler.ServerHandler)
	hc := new(http.HttpCodec)

	// Start serving!
	log.Fatal(gnet.Serve(httpHandler, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore), gnet.WithCodec(hc)), gnet.WithReusePort(true))
}
