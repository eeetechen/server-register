package etcd_grpc

import (
	"context"
	"fmt"
	"github.com/reyukari/server-register/etcd/etcd-grpc/api"
	"github.com/reyukari/server-register/etcd/register"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var ServerName string = "node.srv.app"

type ApiService struct{}

func (a ApiService) ApiTest(ctx context.Context, request *api.Request) (*api.Response, error) {
	fmt.Println(request.String())
	return &api.Response{Output: "OK"}, nil
}

var Addr = "0.0.0.0:8089"

func main() {
	listener, err := net.Listen("tcp", Addr)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}
	// 新建gRPC服务器实例
	grpcServer := grpc.NewServer()
	// 在gRPC服务器注册我们的服务
	var srv = &ApiService{}
	api.RegisterApiServer(grpcServer, srv)

	go func() {
		err = grpcServer.Serve(listener)
		if err != nil {
			panic(err)
		}
	}()
	s, err := register.NewRegister(
		register.SetName(ServerName),
		register.SetAddress(Addr),
		register.SetUsage("v1"),
		register.SetSrv(srv),
		register.SetEtcdConf(clientv3.Config{
			Endpoints:   []string{"127.0.0.1:2379"},
			DialTimeout: time.Second * 5,
		}),
	)
	if err != nil {
		panic(err)
	}
	c := make(chan os.Signal, 1)
	go func() {
		if s.ListenKeepAliveChan() {
			c <- syscall.SIGQUIT
		}
	}()
	fmt.Println("success === > ", Addr)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for a := range c {
		switch a {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			fmt.Println("退出")
			fmt.Println(s.Close())
			return
		default:
			return
		}
	}

}
