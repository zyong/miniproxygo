package main

import (
	"flag"
	"miniproxy"
)

func main() {
	http := flag.String("http", ":8080", "proxy listen addr")
	flag.Parse()
	server := miniproxy.NewServer(*http)
	server.Start()
	select {}
}
