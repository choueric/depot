package depot

import (
	"encoding/binary"
	"errors"
	"net"
	"strconv"
)

const (
	TunnelHelloMsg = "hello server"
	TunnelReplyMsg = "hello local"
)

// request address from socks5
//  +------+----------+----------+
//  | ATYP | BND.ADDR | BND.PORT |
//  +------+----------+----------+
//  |  1   | Variable |    2     |
//  +------+----------+----------+
//
// -  ATYP: address type
//    -  IP V4: 0x01
//    -  DOMAINNAME: 0x03
//    -  IP V6: 0x04
// -  BND.ADDR: host address
// -  BND.PORT: host address port
type ReqAddr struct {
	Atype int
	Host  string
	Port  string
	Raw   []byte
}

func (r *ReqAddr) Address() string {
	return net.JoinHostPort(r.Host, r.Port)
}

func NewReqAddr(raw []byte) (*ReqAddr, error) {
	if len(raw) < 4 {
		return nil, errors.New("socksraw too short")
	}

	var reqAddr ReqAddr

	switch raw[0] {
	case 0x01:
		reqAddr.Atype = int(raw[0])
		reqAddr.Host = net.IP(raw[1 : 1+net.IPv4len]).String()
		port := int(binary.BigEndian.Uint16(raw[1+net.IPv4len:]))
		reqAddr.Port = strconv.Itoa(port)
	case 0x04:
		reqAddr.Atype = int(raw[0])
		reqAddr.Host = net.IP(raw[1 : 1+net.IPv6len]).String()
		port := int(binary.BigEndian.Uint16(raw[1+net.IPv6len:]))
		reqAddr.Port = strconv.Itoa(port)
	case 0x03:
		reqAddr.Atype = int(raw[0])
		reqAddr.Host = string(raw[2 : 2+raw[1]])
		port := int(binary.BigEndian.Uint16(raw[2+raw[1]:]))
		reqAddr.Port = strconv.Itoa(port)
	default:
		return nil, errors.New("invalid atype in socksraw")
	}
	reqAddr.Raw = make([]byte, len(raw))
	copy(reqAddr.Raw, raw)

	return &reqAddr, nil
}
