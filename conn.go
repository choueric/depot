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

//  address request from socks5
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
type AddrReq struct {
	Atype int
	Host  string
	Port  string
	Raw   []byte
}

func (r *AddrReq) Address() string {
	return net.JoinHostPort(r.Host, r.Port)
}

func (r *AddrReq) String() string {
	return r.Address()
}

func NewReqAddr(raw []byte) (*AddrReq, error) {
	if len(raw) < 4 {
		return nil, errors.New("socksraw too short")
	}

	parsePort := func(p []byte) string {
		pp := int(binary.BigEndian.Uint16(p))
		// fix transmission remote's endianess bug
		if pp == 33571 { // 33571=0x8323, 0x2383=9091
			pp = int(binary.LittleEndian.Uint16(p))
		}
		return strconv.Itoa(pp)
	}

	var addrReq AddrReq
	switch raw[0] {
	case 0x01:
		addrReq.Atype = int(raw[0])
		addrReq.Host = net.IP(raw[1 : 1+net.IPv4len]).String()
		addrReq.Port = parsePort(raw[1+net.IPv4len:])
	case 0x04:
		addrReq.Atype = int(raw[0])
		addrReq.Host = net.IP(raw[1 : 1+net.IPv6len]).String()
		addrReq.Port = parsePort(raw[1+net.IPv6len:])
	case 0x03:
		addrReq.Atype = int(raw[0])
		addrReq.Host = string(raw[2 : 2+raw[1]])
		addrReq.Port = parsePort(raw[2+raw[1]:])
	default:
		return nil, errors.New("invalid atype in socksraw")
	}
	addrReq.Raw = make([]byte, len(raw))
	copy(addrReq.Raw, raw)

	return &addrReq, nil
}
