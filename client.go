package server_register

import (
	"context"
	"fmt"
	retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/reyukari/server-register/loadbalence"
	"github.com/reyukari/server-register/proto"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/resolver"
	"google.golang.org/protobuf/types/known/timestamppb"
	"sync"
	"time"
)

var kacp = keepalive.ClientParameters{
	Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
	Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
	PermitWithoutStream: true,             // send pings even without active streams
}

type Client struct {
	hostname     string
	registerAddr string

	currentTaskId int32

	client proto.RegisterClient

	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
}

// NewClient returns the pointer of the Client structure
func NewClient(addr string) (*Client, error) {
	hostname, err := GetHostName()
	if err != nil {
		panic(err)
	}

	r, err := loadbalence.NewUsageLB(
		loadbalence.SetName(hostname),
		loadbalence.SetLoadBalancingPolicy(loadbalence.UsageLB),
		loadbalence.SetEtcdConf(clientv3.Config{
			Endpoints:   []string{"127.0.0.1:2379"},
			DialTimeout: time.Second * 5,
		}))
	if err != nil {
		panic(err)
	}
	resolver.Register(r)

	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		hostname:      hostname,
		registerAddr:  addr,
		currentTaskId: 0,
		ctx:           ctx,
		cancel:        cancel,
	}
	opts := []retry.CallOption{
		retry.WithBackoff(retry.BackoffLinear(100 * time.Millisecond)),
		retry.WithCodes(codes.NotFound, codes.Aborted),
	}
	cc, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, loadbalence.UsageLB)), grpc.WithUnaryInterceptor(retry.UnaryClientInterceptor(opts...)))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	c.client = proto.NewRegisterClient(cc)
	c.register()
	go c.timer()
	return c, nil
}

func (c *Client) Close() {
	c.once.Do(func() {
		c.cancel()
	})
}

func (c *Client) DoneSignal() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Client) timer() {
	defer c.Close()
	tick := time.NewTicker(time.Second * 1)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if err := c.task(); err != nil {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) register() {
	//timestamp, err := ptypes.TimestampProto(time.Now())
	//if err != nil {
	//	panic(err)
	//}
	timestamp := timestamppb.New(time.Now())
	if _, err := c.client.Register(
		context.Background(),
		&proto.RegisterRequest{Hostname: c.hostname, Time: timestamp},
		retry.WithMax(3),
		retry.WithPerRetryTimeout(1*time.Second),
	); err != nil {
		fmt.Println(err)
	}
}

func (c *Client) task() (err error) {
	var reply *proto.PullTaskReply

	if reply, err = c.client.PullTask(
		context.Background(),
		&proto.PullTaskRequest{Hostname: c.hostname},
		retry.WithMax(3),
		retry.WithPerRetryTimeout(1*time.Second),
	); err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("reply:", reply)
	if reply.Task.TaskId == 0 || c.currentTaskId >= reply.Task.TaskId {
		return nil
	}
	c.currentTaskId = reply.Task.TaskId
	if len(reply.Task.Command) == 0 {
		return nil
	}
	var out string
	if out, err = Exec(reply.Task.Command); err != nil {
		fmt.Println(err)
		return err
	}
	if _, err = c.client.CompleteTask(context.Background(), &proto.CompleteTaskRequest{Hostname: c.hostname, TaskId: reply.Task.TaskId, OutPut: out}, retry.WithMax(3), retry.WithPerRetryTimeout(1*time.Second)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
