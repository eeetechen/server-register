package server

import (
	"flag"
	"github.com/reyukari/server-register"
	"github.com/reyukari/server-register/signal"
)

var (
	grpcService string
	httpService string
)

func init() {
	flag.StringVar(&grpcService, "grpcservice", "0.0.0.0:10000", "The address of the grpc server.")
}

func main() {
	flag.Parse()
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signal.SetupSignalHandler()
	server := server_register.NewServer(grpcService)
	<-stopCh
	server.Close()
}
