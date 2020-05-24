package cluster

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/variable"
	"reflect"
	"testing"
	"time"

	//_ "mosn.io/mosn/pkg/stream/http"
	//_ "mosn.io/mosn/pkg/stream/http2"
	"mosn.io/mosn/pkg/types"
)

func Test_maglevLoadBalancer(t *testing.T) {
	hostSet := getMockHostSet()
	lb := newMaglevLoadBalancer(hostSet)

	testProtocol := "SomeProtocol"
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamProtocol, types.ProtocolName(testProtocol))
	lbctx := &mockLbContext{
		context: ctx,
		ch:      &v2.HeaderHashPolicy{Key: "header_key"},
	}

	headerGetter := func(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
		return "test_header_value", nil
	}
	cookieGetter := func(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
		return "test_cookie_value", nil
	}

	testFunc := func(expect []string) {
		hostsResult := []string{}
		// query 5 times
		for i := 0; i < 5; i++ {
			hostsResult = append(hostsResult, lb.ChooseHost(lbctx).Hostname())
		}
		if !reflect.DeepEqual(expect, hostsResult) {
			t.Errorf("hosts expect to be %+v, get %+v", expect, hostsResult)
			t.FailNow()
		}
	}

	// test header
	headerValue := variable.NewBasicVariable("SomeProtocol_request_header_", nil, headerGetter, nil, 0)
	variable.RegisterPrefixVariable(headerValue.Name(), headerValue)
	variable.RegisterProtocolResource(types.ProtocolName(testProtocol), api.HEADER, types.VarProtocolRequestHeader)
	testFunc([]string{
		"host-8", "host-8", "host-8", "host-8", "host-8",
	})

	// test cookie

	cookieValue := variable.NewBasicVariable("SomeProtocol_http_cookie_", nil, cookieGetter, nil, 0)
	variable.RegisterPrefixVariable(cookieValue.Name(), cookieValue)
	variable.RegisterProtocolResource(types.ProtocolName(testProtocol), api.COOKIE, types.VarPrefixHttpCookie)
	lbctx.ch = &v2.HttpCookieHashPolicy{
		Name: "cookie_name",
		Path: "cookie_path",
		TTL:  api.DurationConfig{5 * time.Second},
	}
	testFunc([]string{
		"host-0", "host-0", "host-0", "host-0", "host-0",
	})

	// test source IP
	lbctx.ch = &v2.SourceIPHashPolicy{
	}
	testFunc([]string{
		"host-8", "host-8", "host-8", "host-8", "host-8",
	})
}

func Test_segmentTreeFallback(t *testing.T) {
	hostSet := getMockHostSet()

	mgv := newMaglevLoadBalancer(hostSet)

	// set host-8 unhealthy
	hostSet.hosts[8].SetHealthFlag(types.FAILED_ACTIVE_HC)
	h := hostSet.hosts[8].Health()
	if !assert.Falsef(t, h, "Health() should be false") {
		t.FailNow()
	}
	node, err := mgv.(*maglevLoadBalancer).fallbackSegTree.Leaf(8)
	if err != nil {
		t.Error(err)
	}
	mgv.(*maglevLoadBalancer).fallbackSegTree.Update(node)

	host := mgv.(*maglevLoadBalancer).chooseHostFromSegmentTree(8)
	if !assert.Equalf(t, "host-9", host.Hostname(), "host name should be 'host-9'") {
		t.FailNow()
	}

	// set host-9 unhealthy
	hostSet.hosts[9].SetHealthFlag(types.FAILED_ACTIVE_HC)
	h = hostSet.hosts[9].Health()
	if !assert.Falsef(t, h, "Health() should be false") {
		t.FailNow()
	}
	node, err = mgv.(*maglevLoadBalancer).fallbackSegTree.Leaf(9)
	if err != nil {
		t.Error(err)
	}
	mgv.(*maglevLoadBalancer).fallbackSegTree.Update(node)

	host = mgv.(*maglevLoadBalancer).chooseHostFromSegmentTree(8)
	if !assert.Equalf(t, "host-6", host.Hostname(), "host name should be 'host-6'") {
		t.FailNow()
	}
}

func getMockHostSet() *mockHostSet {
	hosts := []types.Host{}
	hostCount := 10
	for i := 0; i < hostCount; i++ {
		h := &mockHost{
			name: fmt.Sprintf("host-%d", i),
			addr: fmt.Sprintf("127.0.0.%d", i),
		}
		hosts = append(hosts, h)
	}
	return &mockHostSet{
		hosts: hosts,
	}
}
