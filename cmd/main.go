package main

import (
	"flag"
	"gsproxy"
)

func main() {
	http := flag.String("http", ":8080", "proxy listen addr")
	flag.Parse()
	server := gsproxy.NewServer(*http)
	server.Start()
	select {}
}
