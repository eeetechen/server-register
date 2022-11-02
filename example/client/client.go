package main

import (
	"flag"
	server_register "github.com/reyukari/server-register"
	"os"
	"os/signal"
	"syscall"
)

var (
	grpcService string
)

func init() {
	flag.StringVar(&grpcService, "grpcservice", "127.0.0.1:10000", "The address of the grpc server.")
}

func main() {
	flag.Parse()
	// set up signals so we handle the first shutdown signal gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	client, err := server_register.NewClient(grpcService)
	if err != nil {
		panic(err)
	}
	<-c
	client.Close()
	os.Exit(1)
}
