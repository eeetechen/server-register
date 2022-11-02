package loadbalence

import (
	"errors"
	"github.com/reyukari/server-register/etcd/register"
	"go.uber.org/zap"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

const UsageLB = "usageLB"

func newUsageBuilder(opt *Options) {
	//balancer.Builder
	builder := base.NewBalancerBuilder(UsageLB, &ULbPickerBuild{opt: opt}, base.Config{HealthCheck: true})
	balancer.Register(builder)
	return
}

type ULbPickerBuild struct {
	opt *Options
}

func (r *ULbPickerBuild) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	var scs = make(map[balancer.SubConn]*Opt, len(info.ReadySCs))
	for conn, addr := range info.ReadySCs {
		nodeInfo := GetNodeInfo(addr.Address)
		if nodeInfo != nil {
			weight, _ := strconv.Atoi(nodeInfo.Node.Weight)
			scs[conn] = &Opt{
				nodeInfo,
				weight,
			}
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
	node map[balancer.SubConn]*Opt
	mu   sync.Mutex
}

type Opt struct {
	*register.Options
	weight int
}

func (p *ULBPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	t := time.Now().UnixNano() / 1e6
	defer p.mu.Unlock()
	sc := p.RandNode()
	if sc == nil {
		return balancer.PickResult{}, errors.New("no match found conn")
	}
	return balancer.PickResult{SubConn: sc, Done: func(data balancer.DoneInfo) {
		zap.S().Info("test", info.FullMethodName, "end", data.Err, "time", time.Now().UnixNano()/1e6-t)
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

func (p *ULBPicker) RandNode() balancer.SubConn {
	var total int
	for _, nodeOpt := range p.node {
		total += nodeOpt.weight
	}

	rand.Seed(time.Now().UnixNano())
	curr := rand.Intn(total)
	for conn, nodeOpt := range p.node {
		if curr <= nodeOpt.weight {
			return conn
		}
		curr -= nodeOpt.weight
	}

	return nil
}
