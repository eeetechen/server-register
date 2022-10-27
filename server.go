package server_register

import (
	"context"
	"fmt"
	"github.com/reyukari/server-register/example/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

var kaep = keepalive.EnforcementPolicy{
	MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
	PermitWithoutStream: true,            // Allow pings even when there are no active streams
}

var kasp = keepalive.ServerParameters{
	MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
	MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
	MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
	Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
	Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
}

type Server struct {
	ret *Return
	hub *Hub
}

func (s *Server) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterReply, error) {
	if err := s.hub.Register(req.Hostname); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &proto.RegisterReply{}, nil
}

func (s *Server) PullTask(ctx context.Context, req *proto.PullTaskRequest) (*proto.PullTaskReply, error) {
	task := s.hub.PullTask(req.Hostname)
	return &proto.PullTaskReply{Task: &proto.Task{TaskId: task.Id, Command: task.Command}}, nil
}

func (s *Server) CompleteTask(ctx context.Context, req *proto.CompleteTaskRequest) (*proto.CompleteTaskReply, error) {
	if err := s.hub.CompleteTask(req.Hostname, req.TaskId, req.OutPut); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &proto.CompleteTaskReply{}, nil
}

func (s *Server) NewTask(commands []string) {
	s.hub.NewTask(commands)
}

func NewServer(grpcAddr string) *Server {
	hub := NewHub(10)
	s := &Server{
		hub: hub,
	}
	svr := grpc.NewServer()
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		panic(fmt.Sprintf("Failed to listen on addr:", grpcAddr))
	}
	proto.RegisterRegisterServer(svr, s)
	go func() {
		if err := svr.Serve(lis); err != nil {
			panic(err)
		}
	}()
	go func() {
		fmt.Println("new task")
		s.NewTask([]string{"date"})
	}()
	return s
}
