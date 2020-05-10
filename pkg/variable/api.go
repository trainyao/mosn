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
	"mosn.io/mosn/pkg"
	"strings"

	"mosn.io/mosn/pkg/types"
)

func GetVariableValue(ctx context.Context, name string, data ...interface{}) (string, error) {
	// 1. find built-in variables
	if variable, ok := variables[name]; ok {
		// 1.1 check indexed value
		if indexer, ok := variable.(Indexer); ok {
			return getFlushedVariableValue(ctx, indexer.GetIndex())
		}

		// 1.2 use variable.Getter() to get value
		getter := variable.Getter()
		if getter == nil {
			return "", errors.New(errGetterNotFound + name)
		}
		return getter(ctx, nil, variable.Data())
	}

	// 2. find prefix variables
	fullName, ok := data[0].(string)
	if ! ok {
	}

	for prefix, variable := range prefixVariables {
		if strings.HasPrefix(fullName, prefix) {
			fmt.Printf(pkg.TrainLogFormat+"found prefix %s of %s\n", prefix, fullName)
			fmt.Printf(pkg.TrainLogFormat+"var %s\n", variable.Name())

			getter := variable.Getter()
			if getter == nil {
				return "", errors.New(errGetterNotFound + name)
			}
			return getter(ctx, nil, data[0])
		}
	}

	return "", errors.New(errUndefinedVariable + name)
}

func SetVariableValue(ctx context.Context, name, value string) error {
	// find built-in & indexed variables, prefix and non-indexed are not supported
	if variable, ok := variables[name]; ok {
		// 1.1 check indexed value
		if indexer, ok := variable.(Indexer); ok {
			return setFlushedVariableValue(ctx, indexer.GetIndex(), value)
		}
	}

	return errors.New(errSupportIndexedOnly + ": set variable value")
}

// TODO: provide direct access to this function, so the cost of variable name finding could be optimized
func getFlushedVariableValue(ctx context.Context, index uint32) (string, error) {
	if variables := ctx.Value(types.ContextKeyVariables); variables != nil {
		if values, ok := variables.([]IndexedValue); ok {
			value := &values[index]
			if value.Valid || value.NotFound {
				if !value.noCacheable {
					return value.data, nil
				}

				// clear flags
				//value.Valid = false
				//value.NotFound = false
			}

			return getIndexedVariableValue(ctx, value, index)
		}
	}

	return "", errors.New(errNoVariablesInContext)
}

func getIndexedVariableValue(ctx context.Context, value *IndexedValue, index uint32) (string, error) {
	variable := indexedVariables[index]

	//if value.NotFound || value.Valid {
	//	return value.data, nil
	//}

	getter := variable.Getter()
	if getter == nil {
		return "", errors.New(errGetterNotFound + variable.Name())
	}
	vdata, err := getter(ctx, value, variable.Data())
	if err != nil {
		value.Valid = false
		value.NotFound = true
		return vdata, err
	}

	value.data = vdata
	if (variable.Flags() & MOSN_VAR_FLAG_NOCACHEABLE) == 1 {
		value.noCacheable = true
	}
	return value.data, nil
}

func setFlushedVariableValue(ctx context.Context, index uint32, value string) error {
	if variables := ctx.Value(types.ContextKeyVariables); variables != nil {
		if values, ok := variables.([]IndexedValue); ok {
			variable := indexedVariables[index]
			variableValue := &values[index]

			setter := variable.Setter()
			if setter == nil {
				return errors.New(errSetterNotFound + variable.Name())
			}
			return setter(ctx, variableValue, value)
		}
	}

	return errors.New(errNoVariablesInContext)
}
