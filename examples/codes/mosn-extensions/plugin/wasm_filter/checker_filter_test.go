package main

import (
	"context"
	"testing"
)

func Test_test(t *testing.T) {
	f := NewWasmFilter(context.Background(), nil)
	f.test()
}
