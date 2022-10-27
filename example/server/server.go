package server

import (
	"flag"
	"github.com/reyukari/server-register"
	"github.com/reyukari/server-register/signal"
)

var (
	grpcservice string
	httpservice string
)

func init() {
	flag.StringVar(&grpcservice, "grpcservice", "0.0.0.0:10000", "The address of the grpc server.")
	flag.StringVar(&httpservice, "httpservice", "0.0.0.0:10001", "The address of the http server.")
}

func main() {
	flag.Parse()
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signal.SetupSignalHandler()
	server := server_register.NewServer(grpcservice, httpservice)
	<-stopCh
	server.Close()
}
