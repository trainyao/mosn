package http2

import (
	"context"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"testing"
)

func Test_maglevHeaderValue(t *testing.T) {
	header := protocol.CommonHeader(map[string]string{
		"header_key": "header_value",
	})
	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamHeaders, header)
	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamProtocol, protocol.HTTP2)


	MaglevLoadBalancerTemplateForHead



	//variable.RegisterPrefixVariable(types.VarPrefixHttp2Header,
	//	variable.NewBasicVariable(types.VarPrefixHttp2Header, nil, headerGetter, nil, 0))
	//
	//variable.RegisterProtocolResource(protocol.HTTP2, api.HEADER, types.VarPrefixHttp2Header)
	//
	//ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamProtocol, protocol.HTTP2)


}
