package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/perlin-network/life/exec"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/plugin/proto"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream("wasm_plugin", CreateDemoFactory)
}

func CreateDemoFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	args := []string{}
	path, ok := conf["config_path"].(string)
	if ok {
		args = []string{"-c", path}
	}
	client, err := plugin.Register("checker", &plugin.Config{
		Args: args,
	})
	if err != nil {
		return nil, err
	}
	return &factory{
		client: client,
	}, nil
}

type factory struct {
	client *plugin.Client
}

func (f *factory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := NewWasmFilter(ctx, f.client)
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
}

type WasmFilter struct {
	vm  *exec.VirtualMachine
	client  *plugin.Client
	handler api.StreamReceiverFilterHandler
}

func NewWasmFilter(ctx context.Context, client *plugin.Client) *WasmFilter {
	//input, err := ioutil.ReadFile("./filter_main.wasm.bck")
	input, err := ioutil.ReadFile("./filter_main.wasm")
	//input, err := ioutil.ReadFile("./fibonacci.wasm")
	if err != nil {
		panic(err)
	}

	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{}, &exec.NopResolver{}, nil)
	if err != nil {
		panic(err)
	}

	return &WasmFilter{
		vm: vm,
		client: client,
	}
}

func (f *WasmFilter) test() {
	e, found := f.vm.GetFunctionExport("fff")
	if !found {
		panic(fmt.Errorf("not found"))
	}

	v, err := f.vm.Run(e)
	if err != nil {
		panic(err)
	}
	fmt.Println(v)
}

func (f *WasmFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *WasmFilter) OnDestroy() {}

func (f *WasmFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	h := make(map[string]string)
	headers.Range(func(k, v string) bool {
		h[k] = v
		return true
	})
	resp, err := f.client.Call(&proto.Request{
		Header: h,
	}, time.Second)
	if err != nil {
		f.handler.SendHijackReply(500, headers)
		return api.StreamFilterStop
	}
	log.DefaultLogger.Infof("get reposne status:%d", resp.Status)
	if resp.Status == -1 {
		f.handler.SendHijackReply(403, headers)
		return api.StreamFilterStop
	}
	return api.StreamFilterContinue
}
