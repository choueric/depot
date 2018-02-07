package main

import (
	"errors"
	"flag"
	"io"
	"net"
	"strconv"

	"github.com/choueric/clog"
	"github.com/choueric/depot"
)

type controlInfo struct {
	ctrlConn   net.Conn
	tunnelChan chan net.Conn
}

var (
	debug      = true
	dbgLog     = depot.SetDebug(debug)
	configFile = depot.GetDefaultConfigPath()
	config     *depot.Config
	listenAddr string
	ctrlInfo   controlInfo
)

func initialCtrlInfo(conn net.Conn) {
	ctrlInfo.ctrlConn = conn
	ctrlInfo.tunnelChan = make(chan net.Conn)
}

func clearCtrlInfo() {
	ctrlInfo.ctrlConn = nil
	ctrlInfo.tunnelChan = nil
}

func controlHandshake(conn net.Conn) (err error) {
	buf := make([]byte, len(depot.TunnelHelloMsg))
	depot.SetReadTimeout(conn)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	if depot.TunnelHelloMsg != string(buf) {
		return errors.New("tunnel: wrong hello msg")
	}
	conn.Write([]byte(depot.TunnelReplyMsg))

	return nil
}

func handleTunnelConn(conn net.Conn) {
	dbgLog.Println("tunnel connection:", conn.RemoteAddr())
	ctrlInfo.tunnelChan <- conn
}

func listen(host, port, name string) (net.Listener, error) {
	addr := net.JoinHostPort(host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	clog.Printf("start listen %s at %v ...\n", name, addr)
	return ln, nil
}

func serveSocks5(host, port string) {
	socksLn, err := listen(host, port, "socks5")
	if err != nil {
		clog.Fatal("socks5", err)
	}

	for {
		dbgLog.Println("Wait on socks port ...")
		conn, err := socksLn.Accept()
		if err != nil {
			clog.Error("accept socks5:", err)
			continue
		}

		if ctrlInfo.ctrlConn == nil {
			conn.Close()
			clog.Warn("no control connection yet")
			continue
		}

		go handleSocks5Conn(conn)
	}
}

func serveTunnel(tunnelLn net.Listener, done <-chan struct{}) {
	for {
		conn, err := tunnelLn.Accept()
		if err != nil {
			select {
			case <-done:
				dbgLog.Warn("tunnel listener is done")
				return
			}
			clog.Error("tunnel accept:", err)
			continue
		}
		go handleTunnelConn(conn)
	}
}

func serveControl(host, ctrlPort, tunnelPort string) {
	ctrlLn, err := listen(host, ctrlPort, "control")
	if err != nil {
		clog.Fatal("control", err)
	}

	for {
		dbgLog.Println("Wait on control port ...")
		ctrlConn, err := ctrlLn.Accept()
		if err != nil {
			clog.Error("accept control: ", err)
			continue
		}
		dbgLog.Println("control connection:", ctrlConn.RemoteAddr())

		if err := controlHandshake(ctrlConn); err != nil {
			ctrlConn.Close()
			continue
		}

		tunnelLn, err := listen(host, tunnelPort, "tunnel")
		if err != nil {
			clog.Error("listen tunnel", err)
			ctrlConn.Close()
			continue
		}

		initialCtrlInfo(ctrlConn)
		done := make(chan struct{})
		go serveTunnel(tunnelLn, done)

		for {
			buf := make([]byte, len(depot.TunnelAliveMsg))
			_, err := io.ReadFull(ctrlConn, buf)
			if err != nil {
				close(done)
				tunnelLn.Close()
				ctrlConn.Close()
				clearCtrlInfo()
				dbgLog.Warn(ctrlConn.RemoteAddr(), " is dead\n")
				break
			}
		}
	}
}

func init() {
	clog.SetFlags(clog.Ldate | clog.Ltime | clog.Lshortfile | clog.Lcolor)

	flag.StringVar(&configFile, "c", configFile, "specify config file")
	flag.StringVar(&listenAddr, "a", "",
		"local address, listen only to this address if specified")
	flag.Parse()
}

func main() {
	c, err := depot.GetConfig(configFile)
	if err != nil {
		clog.Fatal(err)
	}
	config = c
	dbgLog = depot.SetDebug(config.Debug)
	clog.Println("depot-server")

	go serveControl(listenAddr, strconv.Itoa(config.ControlPort),
		strconv.Itoa(config.TunnelPort))
	serveSocks5(listenAddr, strconv.Itoa(config.ServerPort))
}
