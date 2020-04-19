package http2

import (
	"context"
	"mosn.io/api"
	"mosn.io/mosn/pkg"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"net"
)

var (
//builtinVariables = []variable.Variable{
//	variable.NewBasicVariable(types.VarHttp2RequestPath, nil, requestPathGetter, nil, 0),
//}

//prefixVariables = []variable.Variable{
//	variable.NewBasicVariable(headerPrefix, nil, httpHeaderGetter, nil, 0),
//	variable.NewBasicVariable(argPrefix, nil, httpArgGetter, nil, 0),
//	variable.NewBasicVariable(cookiePrefix, nil, httpCookieGetter, nil, 0),
//}
)

func init() {
	variable.RegisterVariable(variable.NewBasicVariable(types.VarIP, nil, connectionIPGetter, nil, 0))
	variable.RegisterPrefixVariable(types.VarPrefixHttp2Header,
		variable.NewBasicVariable(types.VarPrefixHttp2Header, nil, headerGetter, nil, 0))

	variable.RegisterProtocolResource(protocol.HTTP2, api.IP, types.VarIP)
	variable.RegisterProtocolResource(protocol.HTTP2, api.HEADER, types.VarPrefixHttp2Header)
}

func headerGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (s string, err error) {
	headers, ok := mosnctx.Get(ctx, types.ContextKeyDownStreamHeaders).(api.HeaderMap)
	if !ok {
		return variable.ValueNotFound, nil
	}
	headerKey, ok := data.(string)
	if !ok {
		return variable.ValueNotFound, nil
	}

	log.DefaultLogger.Infof(pkg.TrainLogFormat+" in header getter, headers %+v header key ",headers, headerKey)

	header, found := headers.Get(headerKey)
	if !found {
		return variable.ValueNotFound, nil
	}

	return header, nil
}

func connectionIPGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (s string, err error) {
	remoteAddr := mosnctx.Get(ctx, types.ContextRemoteAddr)
	if remoteAddr != nil {
		log.DefaultLogger.Infof("[train] %s", remoteAddr.(net.Addr).String())
	}

	oriRemoteAddr := mosnctx.Get(ctx, types.ContextOriRemoteAddr)
	if oriRemoteAddr == nil {
		log.DefaultLogger.Infof("[train] ori add is nil")
		return "", nil
	}

	return oriRemoteAddr.(net.Addr).String(), nil
}
