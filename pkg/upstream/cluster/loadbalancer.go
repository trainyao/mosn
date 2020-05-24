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
	"math"
	"math/rand"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/module/segmenttree"
	"mosn.io/mosn/pkg/variable"
	"net"
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

func newMaglevLoadBalancer(set types.HostSet) types.LoadBalancer {
	var table *maglev.Table
	var tree *segmenttree.Tree
	names := []string{}
	for _, host := range set.Hosts() {
		names = append(names, host.Hostname())
	}
	if len(names) != 0 {
		table = maglev.New(names, maglev.SmallM)
		// build tree, devide hash
		nodes := []segmenttree.Node{}
		step := math.MaxUint64 / uint64(len(names))
		for index := range names {
			nodes = append(nodes, segmenttree.Node{
				Value:      index,
				RangeStart: uint64(index) * step,
				RangeEnd:   uint64(index+1) * step,
			})
		}
		updateFunc := func(lv, rv interface{}) interface{} {
			if lv != nil {
				leftIndex, ok := lv.(int)
				if !ok {
					return nil
				}
				if set.Hosts()[leftIndex].Health() {
					return leftIndex
				}
			}

			if rv != nil {
				rightIndex, ok := rv.(int)
				if !ok {
					return nil
				}

				if set.Hosts()[rightIndex].Health() {
					return rightIndex
				}
			}

			return nil
		}

		tree = segmenttree.NewTree(nodes, updateFunc)
	}

	mgv := &maglevLoadBalancer{
		hosts:           set,
		maglev:          table,
		fallbackSegTree: tree,
	}

	return mgv
}

type maglevLoadBalancer struct {
	hosts           types.HostSet
	maglev          *maglev.Table
	fallbackSegTree *segmenttree.Tree
}

func (lb *maglevLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	// host empty, maglev info may be nil
	if lb.maglev == nil {
		return nil
	}

	ch := context.ConsistentHashCriteria()
	if ch == nil || ch.HashType() != api.Maglev {
		return nil
	}

	hash := lb.generateChooseHostHash(context, ch)
	index := lb.maglev.Lookup(hash)
	chosen := lb.hosts.Hosts()[index]

	// fallback
	if !chosen.Health() {
		chosen = lb.chooseHostFromSegmentTree(index)
	}

	log.Proxy.Debugf(nil, "[lb][maglev] get index %d host %s %s",
		index, chosen.Hostname(), chosen.AddressString())

	return chosen
}

func (lb *maglevLoadBalancer) generateChooseHostHash(context types.LoadBalancerContext, info api.ConsistentHashCriteria) uint64 {
	switch info.(type) {
	case *v2.HeaderHashPolicy:
		headerKey := info.(*v2.HeaderHashPolicy).Key
		protocolVarHeaderKey := fmt.Sprintf("%s%s", types.VarProtocolRequestHeader, headerKey)

		headerValue, err := variable.GetProtocolResource(context.DownstreamContext(), api.HEADER, protocolVarHeaderKey)

		if err == nil {
			hashString := fmt.Sprintf("%s:%s", headerKey, headerValue)
			hash := getHashByString(hashString)
			return hash
		}
	case *v2.SourceIPHashPolicy:
		return getHashByAddr(context.DownstreamConnection().RemoteAddr())
	case *v2.HttpCookieHashPolicy:
		info := info.(*v2.HttpCookieHashPolicy)
		cookieName := info.Name
		protocolVarKey := fmt.Sprintf("%s%s", types.VarPrefixHttpCookie, cookieName)

		cookieValue, err := variable.GetProtocolResource(context.DownstreamContext(), api.COOKIE, protocolVarKey)
		if err == nil {
			h := getHashByString(fmt.Sprintf("%s=%s", cookieName, cookieValue))
			return h
		}
	default:
	}

	return 0
}

func (lb *maglevLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.HostNum(metadata) > 0
}

func (lb *maglevLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.HealthyHosts())
}

func (lb *maglevLoadBalancer) chooseHostFromSegmentTree(index int) types.Host {
	if lb.fallbackSegTree == nil {
		return nil
	}

	leaf, err := lb.fallbackSegTree.Leaf(index)
	if err != nil {
		log.DefaultLogger.Errorf("[proxy] [maglev] [segmenttree] find leaf of index %d failed, err:%+v", index, err)
		return nil
	}

	// update tree value when
	lb.fallbackSegTree.Update(leaf)

	// leaf already unhealthy, find parent for it
	leaf = lb.fallbackSegTree.FindParent(leaf)
	var host types.Host
	for {
		if leaf.Value != nil {
			hostIndex, ok := leaf.Value.(int)
			if ok {
				if lb.hosts.Hosts()[hostIndex].Health() {
					host = lb.hosts.Hosts()[hostIndex]
					break
				}
			}
		}

		if leaf.IsRoot() {
			break
		}

		leaf = lb.fallbackSegTree.FindParent(leaf)
	}

	return host
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
// TODO:
// WRR
