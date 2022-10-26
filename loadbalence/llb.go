package loadbalence

import (
	"errors"
	"fmt"
	"github.com/reyukari/server-register/etcd/register"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math/rand"
	"strings"
	"sync"
	"time"
)

const UsageLB = "usageLB"

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

func newVersionBuilder(opt *Options) {
	//balancer.Builder
	builder := base.NewBalancerBuilder(UsageLB, &ULbPickerBuild{opt: opt}, base.Config{HealthCheck: true})
	balancer.Register(builder)
	return
}

type ULbPickerBuild struct {
	opt *Options // discovery Options info
}

func (r *ULbPickerBuild) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	var scs = make(map[balancer.SubConn]*register.Options, len(info.ReadySCs))
	for conn, addr := range info.ReadySCs {
		nodeInfo := GetNodeInfo(addr.Address)
		if nodeInfo != nil {
			scs[conn] = nodeInfo
		}
	}
	if len(scs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	return &ULBPicker{
		node: scs,
	}
}

type ULBPicker struct {
	node map[balancer.SubConn]*register.Options
	mu   sync.Mutex
}

func (p *ULBPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	t := time.Now().UnixNano() / 1e6
	defer p.mu.Unlock()
	version := info.Ctx.Value(UsageLB)
	var subConns []balancer.SubConn
	for conn, node := range p.node {
		if version != "" {
			if node.Node.Version == version.(string) {
				subConns = append(subConns, conn)
			}
		}
	}
	if len(subConns) == 0 {
		return balancer.PickResult{}, errors.New("no match found conn")
	}
	index := rand.Intn(len(subConns))
	sc := subConns[index]
	return balancer.PickResult{SubConn: sc, Done: func(data balancer.DoneInfo) {
		fmt.Println("test", info.FullMethodName, "end", data.Err, "time", time.Now().UnixNano()/1e6-t)
	}}, nil
}

type attrKey struct{}

func SetNodeInfo(addr resolver.Address, hInfo *register.Options) resolver.Address {
	//addr.Attributes = attributes.New()
	addr.Attributes = addr.Attributes.WithValue(attrKey{}, hInfo)
	return addr
}

func GetNodeInfo(attr resolver.Address) *register.Options {
	v := attr.Attributes.Value(attrKey{})
	hi, _ := v.(*register.Options)
	return hi
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
