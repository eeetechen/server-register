package register

import (
	"context"
	"encoding/json"
	"fmt"
	etcd_grpc "github.com/reyukari/server-register/etcd/etcd-grpc"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/cpu"
	"go.etcd.io/etcd/client/v3"
	"strconv"
	"time"
)

type Register struct {
	etcdCli       *clientv3.Client
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	opts          *Options
	name          string
}

func NewRegister(opt ...RegisterOptions) (*Register, error) {
	s := &Register{
		opts: newOptions(opt...),
	}
	var ctx, cancel = context.WithTimeout(context.Background(), time.Duration(s.opts.RegisterTtl)*time.Second)
	defer cancel()
	data, err := json.Marshal(s.opts)
	if err != nil {
		return nil, err
	}
	etcdCli, err := clientv3.New(s.opts.EtcdConf)
	if err != nil {
		return nil, err
	}
	s.etcdCli = etcdCli
	//申请租约
	resp, err := etcdCli.Grant(ctx, s.opts.RegisterTtl)
	if err != nil {
		return s, err
	}
	s.name = fmt.Sprintf("%s/%s", s.opts.Node.Path, s.opts.Node.Id)
	//注册节点
	_, err = etcdCli.Put(ctx, s.name, string(data), clientv3.WithLease(resp.ID))
	if err != nil {
		return s, err
	}

	//续约租约
	s.keepAliveChan, err = etcdCli.KeepAlive(context.Background(), resp.ID)
	if err != nil {
		return s, err
	}
	return s, nil
}

func (s *Register) ListenKeepAliveChan() (isClose bool) {
	for range s.keepAliveChan {
	}
	return true
}

func (s *Register) CrontabUpdate() {
	crontab := cron.New()
	task := func() {
		var ctx, cancel = context.WithTimeout(context.Background(), time.Duration(s.opts.RegisterTtl)*time.Second)
		defer cancel()

		percent, _ := cpu.Percent(time.Second, false)
		if len(percent) <= 0 {
			percent = []float64{80}
		}
		usage := int64(float64(etcd_grpc.DefaultValue) / percent[0])
		value := strconv.FormatInt(usage, 10)

		s.opts.SetWeight(value)

		data, err := json.Marshal(s.opts)
		if err != nil {
			return
		}
		_, err = s.etcdCli.Put(ctx, s.name, string(data))
		if err != nil {
			return
		}
	}
	// 添加定时任务, * * * * * 是 crontab,表示每分钟执行一次
	crontab.AddFunc("* * * * *", task)
	// 启动定时器
	crontab.Start()
}

// Close 注销服务
func (s *Register) Close() error {
	if _, err := s.etcdCli.Delete(context.Background(), s.name); err != nil {
		return err
	}
	return s.etcdCli.Close()
}
