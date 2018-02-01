package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/choueric/clog"
	"github.com/choueric/depot"
)

var (
	debug      = true
	dbgLog     = depot.SetDebug(debug)
	configFile = depot.GetDefaultConfigPath()
)

func waitSignal() {
	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)
	for sig := range sigChan {
		if sig == syscall.SIGHUP {
			// TODO: update configurations
		} else {
			// is this going to happen?
			log.Printf("caught signal %v, exit", sig)
			os.Exit(0)
		}
	}
}

func handShake(server net.Conn) error {
	_, err := server.Write([]byte(depot.TunnelHelloMsg))
	if err != nil {
		return err
	}

	buf := make([]byte, 128)
	n, err := server.Read(buf)
	if err != nil {
		return err
	}
	if n != len(depot.TunnelReplyMsg) && string(buf) != depot.TunnelReplyMsg {
		dbgLog.Printf("receiv handShake msg: %s\n", string(buf))
		return errors.New("invalid tunnel hello msg")
	}

	return nil
}

func getRequest(server net.Conn) (*depot.AddrReq, error) {
	buf := make([]byte, 256)
	n, err := server.Read(buf)
	if err != nil {
		return nil, err
	}
	dbgLog.Println("receive socksraw:", buf[0:n])

	addrReq, err := depot.NewReqAddr(buf[0:n])
	if err != nil {
		return nil, err
	}
	dbgLog.Println("socks request:", addrReq)

	return addrReq, nil
}

func handleRequest(addrReq *depot.AddrReq, server, port string) error {
	tunnelConn, err := net.Dial("tcp", net.JoinHostPort(server, port))
	if err != nil {
		clog.Fatal("error connecting to %s:%s: %v\n", server, port, err)
	}

	closed := false
	defer func() {
		if !closed {
			tunnelConn.Close()
		}
	}()

	// send tunnel handshake
	dbgLog.Println("send tunnel handshake:", addrReq.Raw)
	_, err = tunnelConn.Write(addrReq.Raw)
	if err != nil {
		return err
	}

	// TODO: now request the web
	appConn, err := net.Dial("tcp", addrReq.Address())
	if err != nil {
		clog.Error("error connecting to:", addrReq)
		return err
	}
	defer func() {
		if !closed {
			appConn.Close()
		}
	}()

	go depot.PipeThenClose(tunnelConn, appConn)
	depot.PipeThenClose(appConn, tunnelConn)
	closed = true
	return nil
}

func run(server, port string) {
	dbgLog.Printf("connect to server %s:%s\n", server, port)
	serverConn, err := net.Dial("tcp", server+":"+port)
	if err != nil {
		clog.Fatal(err)
	}
	defer serverConn.Close()

	err = handShake(serverConn)
	if err != nil {
		clog.Fatal("error handshaking with server: ", err)
	}

	for {
		req, err := getRequest(serverConn)
		if err != nil {
			clog.Error("get request from server fail: ", err)
			if err == io.EOF {
				os.Exit(0)
			}
			continue
		}

		go handleRequest(req, server, port)
	}
}

func init() {
	clog.SetFlags(clog.Ldate | clog.Ltime | clog.Lshortfile | clog.Lcolor)
	flag.StringVar(&configFile, "c", configFile, "specify config file")
	flag.Parse()
}

func main() {

	config, err := depot.GetConfig(configFile)
	if err != nil {
		dbgLog.Error("get configuration error: %v\n", err)
		return
	}

	dbgLog = depot.SetDebug(config.Debug)
	dbgLog.Println("depot-local")

	go run(config.ServerAddr, strconv.Itoa(config.TunnelPort))
	waitSignal()
}
