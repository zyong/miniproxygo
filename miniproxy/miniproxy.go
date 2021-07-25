package main

import (
	"flag"

	"github.com/zyong/miniproxygo"
)

func main() {
	http := flag.String("http", ":8080", "proxy listen addr")
	flag.Parse()
	server := miniproxygo.NewServer(*http)
	server.Start()
	select {}
}
