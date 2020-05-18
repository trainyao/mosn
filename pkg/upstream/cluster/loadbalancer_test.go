package cluster

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/dchest/siphash"
)

func Test_test(t *testing.T) {
	hash := siphash.Hash(0xbeefcafebabedead, 0, []byte("1"))
	// 15836395326274861737
	print(hash)

	b := [4]byte{}
	for index := range b {
		b[index] = uint8(255)
	}

	//ip := make(net.IP, 4)
	//binary.BigEndian.PutUint32(ip, nn)
	//return ip

	ip := net.IP(b[0:])

	hash = (uint64(binary.LittleEndian.Uint32(b[0:])) << 16) | uint64(256)

	bb := [8]byte{}
	binary.LittleEndian.PutUint64(bb[0:], hash)

	print(string(bb[0:]))

	//s := ip.String()
	ii := ip.To4()
	print(ii)

	//a := uint64(1)
	//
	//buffer := [64]byte{}
	//s := buffer[0:]
	//
	//binary.LittleEndian.PutUint64(s, a)
	//
	//a = a << 16
	//
	//
	//buffer = [64]byte{}
	//s = buffer[0:]
	//a = a | uint64(65535)
	////print(a)
	//
	//
	//binary.LittleEndian.PutUint64(s, a)
	//
	//print(string(s))
}

func Test_segmentTreeFallbackLogic(t *testing.T) {

}
