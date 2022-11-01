package server_register

import (
	"context"

	"fmt"
	"net"

	etcd_grpc "github.com/reyukari/server-register/etcd/etcd-grpc"
	"github.com/reyukari/server-register/proto"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/cpu"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"

	"strconv"
	"time"
)

type Server struct {
	ret           *Return
	hub           *Hub
	etcdCli       *clientv3.Client
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
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

func (s *Server) CrontabUpdate() {
	crontab := cron.New()
	task := func() {
		var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		hostname, err := GetHostName()
		if err != nil {
			panic(err)
		}

		percent, _ := cpu.Percent(time.Second, false)
		if len(percent) <= 0 {
			percent = []float64{80}
		}
		usage := int64(float64(etcd_grpc.DefaultValue) / percent[0])
		value := strconv.FormatInt(usage, 10)

		_, err = s.etcdCli.Put(ctx, hostname, value)
		if err != nil {
			return
		}
	}
	// 添加定时任务, * * * * * 是 crontab,表示每分钟执行一次
	crontab.AddFunc("* * * * *", task)
	// 启动定时器
	crontab.Start()
}

func NewServer(grpcAddr string) *Server {
	hub := NewHub(10)

	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	s := &Server{
		hub:     hub,
		etcdCli: etcdCli,
	}
	svr := grpc.NewServer()
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		panic(fmt.Sprintf("Failed to listen on addr:", grpcAddr))
	}
	proto.RegisterRegisterServer(svr, s)
	var ctx, cancel = context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()
	resp, err := etcdCli.Grant(ctx, 10)
	if err != nil {
		panic(err)
	}

	hostname, err := GetHostName()
	if err != nil {
		panic(err)
	}

	percent, _ := cpu.Percent(time.Second, false)
	if len(percent) <= 0 {
		percent = []float64{80}
	}
	usage := int64(float64(etcd_grpc.DefaultValue) / percent[0])
	value := strconv.FormatInt(usage, 10)
	//注册节点
	_, err = etcdCli.Put(ctx, hostname, string(value), clientv3.WithLease(resp.ID))
	if err != nil {
		panic(err)
	}

	//续约租约
	s.keepAliveChan, err = etcdCli.KeepAlive(context.Background(), resp.ID)
	if err != nil {
		panic(err)
	}

	go func() {
		if err := svr.Serve(lis); err != nil {
			panic(err)
		}
	}()
	go func() {
		fmt.Println("new task")
		s.NewTask([]string{"date"})
	}()

	go s.CrontabUpdate()

	return s
}
