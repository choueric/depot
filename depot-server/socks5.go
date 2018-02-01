package main

import (
	"bytes"
	"errors"
	"io"
	"net"

	"github.com/choueric/clog"
	"github.com/choueric/depot"
)

const (
	socksVer5       = 5
	socksCmdConnect = 1

	VER             = 0
	NMETHODS        = 1
	METHODS         = 2
	ULEN            = 1
	UNAME           = 2
	METHOD_NONE     = 0x00
	METHOD_GSSAPI   = 0x01
	METHOD_USERNAME = 0x02
	METHOD_IANA     = 0x03 // 0x03 - 0x7f
	METHOD_RSV      = 0x80 // 0x80 - 0xfe, reserve for private methods
	METHOD_DENY     = 0xff // no acceptable methods
	CMD             = 1
	ATYP            = 3 // address type index
	DST_ADDR        = 4 // ip addres start index
	DOMAIN_LEN      = 4 // domain address length index
	DOMAIN_ADDR     = 5 // domain address start index
	ATYP_IPV4       = 1 // type is ipv4 address
	ATYP_DOMAIN     = 3 // type is domain address
	ATYP_IPV6       = 4 // type is ipv6 address
)

var (
	errVer           = errors.New("socks version not supported")
	errMethod        = errors.New("socks only support user/password method now")
	errAuthExtraData = errors.New("socks authentication get extra data")
	errAuth          = errors.New("socks invalid username/password")
	errReqExtraData  = errors.New("socks request get extra data")
	errCmd           = errors.New("socks only support CONNECT request")
	errAddrType      = errors.New("socks invalid address type")
)

/*
client:
  +----+----------+----------+
  |VER | NMETHODS | METHODS  |
  +----+----------+----------+
  | 1  |    1     | 1 to 255 |
  +----+----------+----------+
server:
  +----+--------+
  |VER | METHOD |
  +----+--------+
  | 1  |   1    |
  +----+--------+
*/
func socksHandShake(conn net.Conn) (err error) {
	buf := make([]byte, 258)
	depot.SetReadTimeout(conn)

	var n int
	// make sure we get the nmethod field
	if n, err = io.ReadAtLeast(conn, buf, NMETHODS+1); err != nil {
		return err
	}
	dbgLog.Printf("read %v bytes\n", buf[0:n])

	if buf[VER] != socksVer5 {
		return errVer
	}

	nmethod := int(buf[NMETHODS])
	msgLen := nmethod + 2
	if n == msgLen { // done, common case
		// do nothing, jump directly to send confirmation
	} else if n < msgLen { // has more methods to read, rare case
		if _, err = io.ReadFull(conn, buf[n:msgLen]); err != nil {
			return
		}
	} else { // error, should not get extra data
		return errAuthExtraData
	}

	m := METHOD_DENY
	for i := METHODS; i < msgLen; i++ {
		if int(buf[i]) == METHOD_USERNAME {
			m = METHOD_USERNAME
			break
		}
	}

	// send confirmation: version 5,
	_, err = conn.Write([]byte{socksVer5, byte(m)})
	if m == METHOD_DENY {
		// authentication dosen't match
		return errMethod
	}
	return nil
}

/*
user/password sub-authentication
+----+------+----------+------+----------+
|VER | ULEN |   UNAME  | PLEN |  PASSWD  |
+----+------+----------+------+----------+
| 1  |   1  | 1 to 255 |  1   | 1 to 255 |
+----+------+----------+------+----------+
*/
func socksAuthticate(conn net.Conn) (err error) {
	buf := make([]byte, 257) // 255 + 2
	depot.SetReadTimeout(conn)

	if _, err = io.ReadFull(conn, buf[0:2]); err != nil {
		return err
	}

	if buf[VER] != 0x01 {
		return errors.New("user/password sub-auth: invalid version")
	}

	ulen := int(buf[ULEN])
	if _, err = io.ReadFull(conn, buf[0:ulen]); err != nil {
		return
	}
	username := string(buf[0:ulen])
	dbgLog.Println("username:", username)

	if _, err = io.ReadFull(conn, buf[0:1]); err != nil {
		return err
	}
	plen := int(buf[0])
	if _, err = io.ReadFull(conn, buf[0:plen]); err != nil {
		return
	}
	password := string(buf[0:plen])
	dbgLog.Println("password:", password)

	if username != config.UserName || password != config.Password {
		_, err = conn.Write([]byte{socksVer5, 0x01})
		return errAuth
	}
	_, err = conn.Write([]byte{socksVer5, 0x00})
	return nil
}

/*
   +----+-----+-------+------+----------+----------+
   |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
   +----+-----+-------+------+----------+----------+
   | 1  |  1  | X'00' |  1   | Variable |    2     |
   +----+-----+-------+------+----------+----------+

   +----+-----+-------+------+----------+----------+
   |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
   +----+-----+-------+------+----------+----------+
   | 1  |  1  | X'00' |  1   | Variable |    2     |
   +----+-----+-------+------+----------+----------+
*/
func getSocksRequest(conn net.Conn) (reqAddr *depot.ReqAddr, err error) {
	buf := make([]byte, 263)
	var n int
	depot.SetReadTimeout(conn)
	// read till we get possible domain length field
	if n, err = io.ReadAtLeast(conn, buf, DOMAIN_LEN+1); err != nil {
		return
	}
	dbgLog.Printf("read %v bytes\n", buf[0:n])

	if buf[VER] != socksVer5 {
		err = errVer
		return
	}

	if buf[CMD] != socksCmdConnect { // only support CONNECT reqeust now
		err = errCmd
		return
	}

	msgLen := -1
	switch buf[ATYP] {
	case ATYP_IPV4:
		msgLen = 6 + net.IPv4len
	case ATYP_IPV6:
		msgLen = 6 + net.IPv6len
	case ATYP_DOMAIN:
		msgLen = 7 + int(buf[DOMAIN_LEN])
	default:
		err = errAddrType
		return
	}

	if n < msgLen {
		if _, err = io.ReadFull(conn, buf[n:msgLen]); err != nil {
			return
		}
	} else if n > msgLen {
		err = errReqExtraData
		return
	}

	reqAddr, err = depot.NewReqAddr(buf[ATYP:msgLen])
	if err != nil {
		return
	}

	// Sending connection established message immediately to client.
	// This some round trip time for creating socks connection with the client.
	// But if connection failed, the client will get connection reset error.
	// TODO: BND.PORT(0x08, 0x43) is not the actual port number
	reply := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43}
	if _, err = conn.Write(reply); err != nil {
		clog.Error("send connection confirmation:", err)
		return
	}

	return
}

// dialTunnel sends request address to local and waits for local's new tunnel
// connection, shakes hand with it.
func dialTunnel(ctrlConn net.Conn, reqAddr *depot.ReqAddr) (net.Conn, error) {
	if _, err := ctrlConn.Write(reqAddr.Raw); err != nil {
		return nil, err
	}
	c := <-tunnelChan
	dbgLog.Println("get new tunnel connection:", c.RemoteAddr())

	// receive and check the socksraw handshake msg
	// FIXME: If there are many new tunnels, socksraw may not match
	buf := make([]byte, 256)
	n, err := io.ReadAtLeast(c, buf, 5)
	if err != nil {
		return nil, err
	}
	dbgLog.Println("receive handshake of local:", buf[0:n])

	if !bytes.Equal(buf[0:n], reqAddr.Raw) {
		return nil, errors.New("socksraw does not match")
	}
	return c, nil
}

func handleSocks5Conn(socksConn net.Conn) (err error) {
	dbgLog.Printf("socks connect from %s\n", socksConn.RemoteAddr().String())

	closed := false
	defer func() {
		if !closed {
			socksConn.Close()
		}
	}()

	if err = socksHandShake(socksConn); err != nil {
		clog.Error("socks handshake: ", err)
		return
	}

	if err = socksAuthticate(socksConn); err != nil {
		clog.Error("socks authticate:", err)
		return
	}

	reqAddr, err := getSocksRequest(socksConn)
	if err != nil {
		clog.Error("error getting request:", err)
		return
	}
	dbgLog.Printf("request address: %v\n", reqAddr.Address())

	// handle the request to local
	tunnelConn, err := dialTunnel(controlConn, reqAddr)
	if err != nil {
		clog.Error("Failed connect to local")
		return
	}
	defer func() {
		if !closed {
			tunnelConn.Close()
		}
	}()

	go depot.PipeThenClose(socksConn, tunnelConn)
	depot.PipeThenClose(tunnelConn, socksConn)
	closed = true
	dbgLog.Println("closed connection to", reqAddr.Address())
	return nil
}
