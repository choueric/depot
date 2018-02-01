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

var (
	debug       = true
	dbgLog      = depot.SetDebug(debug)
	configFile  = depot.GetDefaultConfigPath()
	config      *depot.Config
	listenAddr  string
	controlConn net.Conn
	tunnelChan  chan net.Conn
)

func controlHandshake(conn net.Conn) (err error) {
	dbgLog.Printf("tunnel connection: %v\n", conn.RemoteAddr())

	buf := make([]byte, len(depot.TunnelHelloMsg))
	depot.SetReadTimeout(conn)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	if depot.TunnelHelloMsg != string(buf) {
		return errors.New("tunnel: wrong hello msg")
	}
	conn.Write([]byte(depot.TunnelReplyMsg))

	controlConn = conn
	tunnelChan = make(chan net.Conn)
	return nil
}

func handleTunnelConn(conn net.Conn) {
	dbgLog.Println("send new sublocal:", conn.RemoteAddr())
	tunnelChan <- conn
}

func serveSocks5(host, port string) {
	addr := net.JoinHostPort(host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		clog.Fatal(err)
	}
	clog.Printf("starting socks5 server at %v ...\n", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			clog.Error("accept socks5:", err)
			continue
		}
		if controlConn == nil {
			conn.Close()
			clog.Warn("no control connection yet")
			continue
		}
		go handleSocks5Conn(conn)
	}
}

func serveTunnel(host, port string) {
	addr := net.JoinHostPort(host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		clog.Fatal(err)
	}
	clog.Printf("starting tunnel server at %v ...\n", addr)

	// accept the control connection
	ctrlConn, err := ln.Accept()
	if err != nil {
		clog.Fatal("accept control: ", err)
	}
	defer ctrlConn.Close()

	if err := controlHandshake(ctrlConn); err != nil {
		clog.Fatal("fail to handshake with local ", err)
	}

	for {
		// accept tunnel connections
		conn, err := ln.Accept()
		if err != nil {
			clog.Error("accept:", err)
			continue
		}
		go handleTunnelConn(conn)
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

	go serveTunnel(listenAddr, strconv.Itoa(config.TunnelPort))
	serveSocks5(listenAddr, strconv.Itoa(config.ServerPort))
}
