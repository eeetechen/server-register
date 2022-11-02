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
	flag.StringVar(&grpcService, "grpcservice", "0.0.0.0:10000", "The address of the grpc server.")
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	server_register.NewServer(grpcService)
	<-c
	os.Exit(1)
}
