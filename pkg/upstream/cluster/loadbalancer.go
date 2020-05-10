/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cluster

import (
	"encoding/binary"
	"fmt"
	"github.com/dchest/siphash"
	"math/rand"
	"mosn.io/mosn/pkg"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/variable"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trainyao/go-maglev"
	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

// NewLoadBalancer can be register self defined type
var lbFactories map[types.LoadBalancerType]func(types.HostSet) types.LoadBalancer

func RegisterLBType(lbType types.LoadBalancerType, f func(types.HostSet) types.LoadBalancer) {
	if lbFactories == nil {
		lbFactories = make(map[types.LoadBalancerType]func(types.HostSet) types.LoadBalancer)
	}
	lbFactories[lbType] = f
}

var rrFactory *roundRobinLoadBalancerFactory

func init() {
	rrFactory = &roundRobinLoadBalancerFactory{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	RegisterLBType(types.RoundRobin, rrFactory.newRoundRobinLoadBalancer)
	RegisterLBType(types.Random, newRandomLoadBalancer)
	RegisterLBType(types.Maglev, newMaglevLoadBalancer)
}

func NewLoadBalancer(lbType types.LoadBalancerType, hosts types.HostSet) types.LoadBalancer {
	if f, ok := lbFactories[lbType]; ok {
		return f(hosts)
	}
	return rrFactory.newRoundRobinLoadBalancer(hosts)
}

// LoadBalancer Implementations

type randomLoadBalancer struct {
	mutex sync.Mutex
	rand  *rand.Rand
	hosts types.HostSet
}

func newRandomLoadBalancer(hosts types.HostSet) types.LoadBalancer {
	return &randomLoadBalancer{
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
		hosts: hosts,
	}
}

func (lb *randomLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	targets := lb.hosts.HealthyHosts()
	if len(targets) == 0 {
		return nil
	}
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	idx := lb.rand.Intn(len(targets))
	return targets[idx]
}

func (lb *randomLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *randomLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

type roundRobinLoadBalancer struct {
	hosts   types.HostSet
	rrIndex uint32
}

type roundRobinLoadBalancerFactory struct {
	mutex sync.Mutex
	rand  *rand.Rand
}

func (f *roundRobinLoadBalancerFactory) newRoundRobinLoadBalancer(hosts types.HostSet) types.LoadBalancer {
	var idx uint32
	hostsList := hosts.Hosts()
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if len(hostsList) != 0 {
		idx = f.rand.Uint32() % uint32(len(hostsList))
	}
	return &roundRobinLoadBalancer{
		hosts:   hosts,
		rrIndex: idx,
	}
}
func (lb *roundRobinLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	targets := lb.hosts.HealthyHosts()
	if len(targets) == 0 {
		return nil
	}
	index := atomic.AddUint32(&lb.rrIndex, 1) % uint32(len(targets))
	return targets[index]
}

func (lb *roundRobinLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *roundRobinLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

func NewMaglevInfo(config *v2.LBMaglevConfig) (i types.LBMaglevInfo) {
	switch config.Type {
	case v2.MaglevType_header:
		if f, ok := config.HashFactor.(*v2.HeaderMaglevHashFactor); !ok {
			log.Proxy.Warnf(nil, "header %s and hash factor not match", config.Type)
			return nil
		} else {
			i = &types.LBHeaderMaglevInfo{Key: f.HeaderKey}
		}
	case v2.MaglevType_http_cookie:
		if f, ok := config.HashFactor.(*v2.HttpCookieMaglevHashFactor); !ok {
			log.Proxy.Warnf(nil, "header %s and hash factor not match", config.Type)
			return nil
		} else {
			i = &types.LBHttpCookieMaglevInfo{
				Name: f.CookieName,
				Path: f.CookiePath,
				TTL:  f.TTL,
			}
		}
	case v2.MaglevType_source_IP:
		if _, ok := config.HashFactor.(*v2.SourceIPMaglevHashFactor); !ok {
			log.Proxy.Warnf(nil, "header %s and hash factor not match", config.Type)
			return nil
		} else {
			i = &types.LBSourceIPMaglevInfo{}
		}
	default:
		log.Proxy.Warnf(nil, "not found maglev factor type %s", config.Type)
	}

	return nil
}

func newMaglevLoadBalancer(set types.HostSet) types.LoadBalancer {
	log.DefaultLogger.Infof(pkg.TrainLogFormat+"in new %s", string(debug.Stack()))

	hosts := []string{}
	for _, h := range set.HealthyHosts() {
		hosts = append(hosts, h.AddressString())
	}

	log.DefaultLogger.Infof("[train] in new %s", strings.Join(hosts, ","))

	healthyHosts := set.HealthyHosts()
	names := []string{}

	var table *maglev.Table
	for _, host := range healthyHosts {
		names = append(names, host.Hostname())
	}
	if len(names) != 0 {
		table = maglev.New(names, maglev.SmallM)
	}

	return &maglevLoadBalancer{
		hosts:        set,
		healthyHosts: healthyHosts,
		maglev:       table,
		rand:         rand.NewSource(int64(time.Now().Nanosecond())),
	}
}

type maglevLoadBalancer struct {
	// TODO train 确定 host 变化会不会动态变化 hostset.healthyhosts
	mutex        sync.Mutex
	hosts        types.HostSet
	healthyHosts []types.Host
	maglev       *maglev.Table
	rand         rand.Source
}

func (lb *maglevLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	log.DefaultLogger.Infof(pkg.TrainLogFormat+"in choose")
	// host empty, maglev info may be nil
	if lb.maglev == nil {
		log.DefaultLogger.Infof(pkg.TrainLogFormat+"lb mgv info == nil")
		return nil
	}

	ch := context.ConsistentHashCriteria()
	if ch == nil || ch.HashType() != api.Maglev {
		log.DefaultLogger.Infof(pkg.TrainLogFormat+"ch != mgv info ")
		return nil
	}

	c, ok := ch.(types.LBMaglevInfo)
	if !ok {
		log.DefaultLogger.Infof(pkg.TrainLogFormat+"type not mgv info")
		return nil
	}

	hash := lb.generateChooseHostHash(context, c)

	// TODO train 确定 长连接 upstream 变化
	// TODO train 确定 upstream 下架之后变化

	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	if len(lb.healthyHosts) == 0 {
		return nil
	}

	index := lb.maglev.Lookup(hash)

	log.Proxy.Infof(nil, "[lb][maglev][train] get index %d host %s %s",
		index, lb.healthyHosts[index].Hostname(), lb.healthyHosts[index].AddressString())

	return lb.healthyHosts[index]
}

func (lb *maglevLoadBalancer) generateChooseHostHash(context types.LoadBalancerContext, info types.LBMaglevInfo) uint64 {
	switch info.(type) {
	case *types.LBHeaderMaglevInfo:
		log.DefaultLogger.Infof("[train] generate header hash")

		headerKey := info.(*types.LBHeaderMaglevInfo).Key
		protocolVarHeaderKey := fmt.Sprintf("%s%s", types.VarProtocolRequestHeader, headerKey)
		log.DefaultLogger.Infof("[train] header key %s", protocolVarHeaderKey)

		headerValue, err := variable.GetProtocolResource(context.DownstreamContext(), api.HEADER, protocolVarHeaderKey)
		log.DefaultLogger.Infof(pkg.TrainLogFormat+"header value %s", headerValue)

		if err == nil {
			hashString := fmt.Sprintf("%s:%s", headerKey, headerValue)
			hash := getHashByString(hashString)
			return hash
		} else {
			log.DefaultLogger.Infof(pkg.TrainLogFormat+ "%+v", err)
		}
	case *types.LBSourceIPMaglevInfo:
		log.DefaultLogger.Infof("[train] generate ip hash")

		return getHashByAddr(context.DownstreamConnection().RemoteAddr())
	case *types.LBHttpCookieMaglevInfo:
		log.DefaultLogger.Infof("[train] generate cookie hash")
		info := info.(*types.LBHttpCookieMaglevInfo)
		cookieName := info.Name
		//cookiePath := info.Path
		//cookieTTL := info.TTL
		//dsn := fmt.Sprintf("cookie://%s/%s?ttl=%s", cookiePath, cookieName, cookieTTL)
		dsn := fmt.Sprintf("%s%s", types.VarPrefixHttpCookie, cookieName)

		log.DefaultLogger.Infof(pkg.TrainLogFormat+"cookie dsn %s", dsn)

		cookieValue, err := variable.GetProtocolResource(context.DownstreamContext(), api.COOKIE, dsn)
		log.DefaultLogger.Infof(pkg.TrainLogFormat+"cookie value %s", cookieValue)
		if err == nil {
			h := getHashByString(fmt.Sprintf("%s&value=%s", cookieValue))
			return h
		}
	default:
		log.DefaultLogger.Infof("[train] generate default hash")
	}

	log.DefaultLogger.Infof("[train] generate random hash")
	return 0
}

func (lb *maglevLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.HostNum(metadata) > 0
}

func (lb *maglevLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.healthyHosts)
}

func getHashByAddr(addr net.Addr) (hash uint64) {
	if tcpaddr, ok := addr.(*net.TCPAddr); ok {
		if len(tcpaddr.IP) == 16 || len(tcpaddr.IP) == 4 {
			var tmp uint32

			if len(tcpaddr.IP) == 16 {
				tmp = binary.BigEndian.Uint32(tcpaddr.IP[12:16])
			} else {
				tmp = binary.BigEndian.Uint32(tcpaddr.IP)
			}
			hash = uint64(tmp)

			return
		}
	}

	return getHashByString(fmt.Sprintf("%s", addr.String()))
}

func getHashByString(str string) uint64 {
	return siphash.Hash(0xbeefcafebabedead, 0, []byte(str))
}

//
//func getRandomHash(source rand.Source) uint64 {
//	return rand.NewSource(int64(time.Now().Nanosecond()))
//}

// TODO:
// WRR
