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

func getRequest(server net.Conn) (*depot.ReqAddr, error) {
	buf := make([]byte, 256)
	n, err := server.Read(buf)
	if err != nil {
		return nil, err
	}
	dbgLog.Println("receive socksraw:", buf[0:n])

	reqAddr, err := depot.NewReqAddr(buf[0:n])
	if err != nil {
		return nil, err
	}
	dbgLog.Println("socks request:", reqAddr)

	return reqAddr, nil
}

func handleRequest(reqAddr *depot.ReqAddr, server, port string) error {
	peer, err := net.Dial("tcp", server+":"+port)
	if err != nil {
		clog.Fatal("error connecting to %s:%s: %v\n", server, port, err)
	}

	closed := false
	defer func() {
		if !closed {
			peer.Close()
		}
	}()

	dbgLog.Println("send socksraw:", reqAddr.Raw)
	_, err = peer.Write(reqAddr.Raw)
	if err != nil {
		return err
	}

	// TODO: now request the web
	app, err := net.Dial("tcp", reqAddr.Address())
	if err != nil {
		clog.Error("error connecting to:", reqAddr)
		return err
	}
	defer func() {
		if !closed {
			app.Close()
		}
	}()

	go depot.PipeThenClose(peer, app)
	depot.PipeThenClose(app, peer)
	closed = true
	return nil
}

func run(server, port string) {
	dbgLog.Printf("connect to server %s:%s\n", server, port)
	peer, err := net.Dial("tcp", server+":"+port)
	if err != nil {
		clog.Fatal(err)
	}

	err = handShake(peer)
	if err != nil {
		clog.Fatal("error handshaking with server: ", err)
	}

	for {
		req, err := getRequest(peer)
		if err != nil {
			clog.Error("get request from tunnel fail: ", err)
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
