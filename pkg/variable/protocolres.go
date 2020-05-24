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

package variable

import (
	"context"
	"errors"
	"fmt"
	"mosn.io/api"
	"mosn.io/mosn/pkg"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
)

var (
	errUnregisterProtocolResource = "unregister Protocol resource, Protocol: "
	protocolVar                   map[string]string
)

func init() {
	protocolVar = make(map[string]string)
}

// RegisterProtocolResource registers the resource as ProtocolResourceName
// forexample protocolVar[Http1+api.URI] = http_request_uri var
func RegisterProtocolResource(protocol types.ProtocolName, resource api.ProtocolResourceName, varname string) error {
	pr := convert(protocol, resource)
	if _, ok := protocolVar[pr]; ok {
		return errors.New("protocol resource already exists, name: " + pr)
	}

	protocolVar[pr] = fmt.Sprintf("%s_%s", protocol, varname)

	return nil
}

// GetProtocolResource get URI,PATH,ARG var depends on ProtocolResourceName
func GetProtocolResource(ctx context.Context, name api.ProtocolResourceName, data ...interface{}) (string, error) {
	p, ok := mosnctx.Get(ctx, types.ContextKeyDownStreamProtocol).(types.ProtocolName)
	if !ok {
		return "", errors.New("get ContextKeyDownStreamProtocol failed.")
	}

	fmt.Printf(pkg.TrainLogFormat+"protocol %s", p)

	n := convert(p, name)
	fmt.Printf(pkg.TrainLogFormat+"n %s", n)

	if v, ok := protocolVar[convert(p, name)]; ok {
		fmt.Printf(pkg.TrainLogFormat+"v %s", v)
		fmt.Printf(pkg.TrainLogFormat+"d %s", data[0])
		//return GetVariableValue(ctx, fmt.Sprintf("%s_%s", p, v), data[0])
		return GetVariableValue(ctx, v, fmt.Sprintf("%s_%s", p, data[0]))
	} else {
		return "", errors.New(errUnregisterProtocolResource + string(p))
	}
}

func convert(p types.ProtocolName, name api.ProtocolResourceName) string {
	return string(p) + string(name)
}
