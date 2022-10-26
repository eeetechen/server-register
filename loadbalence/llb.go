package loadbalence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/reyukari/server-register/etcd/register"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
	"log"
	"strings"
	"sync"
	"time"
)

var ErrLoadBalancingPolicy = errors.New("LoadBalancingPolicy is empty or not apply")

type NodeArray struct {
	Node []register.Options `json:"node"`
}

type UsageLBConf struct {
	etcdCli *clientv3.Client
	cc      resolver.ClientConn
	Node    sync.Map
	opts    *Options
}

func NewUsageLB(opt ...ClientOptions) (resolver.Builder, error) {
	s := &UsageLBConf{
		opts: newOptions(opt...),
	}
	if s.opts.LoadBalancingPolicy == UsageLB {
		newUsageBuilder(s.opts)
	} else {
		return nil, ErrLoadBalancingPolicy
	}
	etcdCli, err := clientv3.New(s.opts.EtcdConf)
	if err != nil {
		return nil, err
	}
	s.etcdCli = etcdCli
	return s, nil
}

// Build 当调用`grpc.Dial()`时执行
func (d *UsageLBConf) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	d.cc = cc
	var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := d.etcdCli.Get(ctx, d.opts.SrvName, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	for _, v := range res.Kvs {
		if err = d.AddNode(v.Key, v.Value); err != nil {
			log.Println(err)
			continue
		}
	}
	if len(res.Kvs) == 0 {
		log.Printf("no %s service found , waiting for the service to join \n", d.opts.SrvName)
	}
	go func(dd *UsageLBConf) {
		dd.watcher()
	}(d)
	return d, err
}

func (d *UsageLBConf) AddNode(key, val []byte) error {
	var data = new(register.Options)
	err := json.Unmarshal(val, data)
	if err != nil {
		return err
	}
	addr := resolver.Address{Addr: data.Node.Address}
	addr = SetNodeInfo(addr, data)
	d.Node.Store(string(key), addr)
	d.cc.UpdateState(resolver.State{Addresses: d.GetAddress()})
	return nil
}

func (d *UsageLBConf) DelNode(key []byte) error {
	keyStr := string(key)
	d.Node.Delete(keyStr)
	d.cc.UpdateState(resolver.State{Addresses: d.GetAddress()})
	return nil
}

func (d *UsageLBConf) GetAddress() []resolver.Address {
	var addr []resolver.Address
	d.Node.Range(func(key, value interface{}) bool {
		addr = append(addr, value.(resolver.Address))
		return true
	})
	return addr
}

func (d *UsageLBConf) Scheme() string {
	return "usageLB"
}

// watcher 监听前缀
func (d *UsageLBConf) watcher() {
	rch := d.etcdCli.Watch(context.Background(), d.opts.SrvName, clientv3.WithPrefix())
	for res := range rch {
		for _, ev := range res.Events {
			switch ev.Type {
			case mvccpb.PUT: //新增或修改
				if err := d.AddNode(ev.Kv.Key, ev.Kv.Value); err != nil {
					log.Println(err)
				}
			case mvccpb.DELETE: //删除
				if err := d.DelNode(ev.Kv.Key); err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func (s *UsageLBConf) ResolveNow(rn resolver.ResolveNowOptions) {
	//log.Println("ResolveNow")
}

func (s *UsageLBConf) Close() {
	s.etcdCli.Close()
}

/*
Conf
*/

type Options struct {
	EtcdConf            clientv3.Config `json:"-"`
	SrvName             string
	LoadBalancingPolicy string
}

type ClientOptions func(*Options)

func SetName(name string) ClientOptions {
	return func(option *Options) {
		option.SrvName = fmt.Sprintf("/%s", strings.Replace(name, ".", "/", -1))
	}
}

func SetEtcdConf(conf clientv3.Config) ClientOptions {
	return func(options *Options) {
		options.EtcdConf = conf
	}
}

func SetLoadBalancingPolicy(name string) ClientOptions {
	return func(options *Options) {
		options.LoadBalancingPolicy = name
	}
}

func newOptions(opt ...ClientOptions) *Options {
	opts := &Options{
		EtcdConf: clientv3.Config{
			Endpoints:   []string{"127.0.0.1:2379"},
			DialTimeout: 5 * time.Second,
		},
	}
	for _, o := range opt {
		o(opts)
	}
	return opts
}
