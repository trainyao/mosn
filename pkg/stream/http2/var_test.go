package http2

import (
	"context"
	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"testing"
)

func Test_maglevHeaderValue(t *testing.T) {
	variable.RegisterPrefixVariable(types.VarPrefixHttp2Header,
		variable.NewBasicVariable(types.VarPrefixHttp2Header, nil, headerGetter, nil, 0))

	variable.RegisterProtocolResource(protocol.HTTP2, api.HEADER, types.VarPrefixHttp2Header)

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyDownStreamProtocol, protocol.HTTP2)


}
