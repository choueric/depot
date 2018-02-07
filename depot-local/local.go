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
	"time"

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

func getRequest(ctrlConn net.Conn) (*depot.AddrReq, error) {
	buf := make([]byte, 256)
	n, err := ctrlConn.Read(buf)
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
	addr := net.JoinHostPort(server, port)
	tunnelConn, err := net.Dial("tcp", addr)
	if err != nil {
		clog.Error("dial tunnel %v error: %v\n", addr, err)
		return err
	}

	closed := false
	defer func() {
		if !closed {
			tunnelConn.Close()
		}
	}()

	dbgLog.Println("send tunnel handshake:", addrReq.Raw)
	_, err = tunnelConn.Write(addrReq.Raw)
	if err != nil {
		return err
	}

	appConn, err := net.Dial("tcp", addrReq.Address())
	if err != nil {
		clog.Error("dial app %v error: \n", addrReq, err)
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
	dbgLog.Println("closed connection to", addrReq)
	return nil
}

func sayAlive(ctrlConn net.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			ctrlConn.Write([]byte(depot.TunnelAliveMsg))
		}
	}
}

func run(server, ctrlPort, tunnelPort string) {
	addr := net.JoinHostPort(server, ctrlPort)
	for {
		dbgLog.Printf("try to connect server ... ")
		ctrlConn, err := net.Dial("tcp", addr)
		if err != nil {
			dbgLog.Warn(err)
			time.Sleep(2 * time.Second)
			continue
		}
		dbgLog.Printf("done via %v\n", ctrlConn.LocalAddr())

		err = handShake(ctrlConn)
		if err != nil {
			clog.Error("error handshaking: ", err)
			ctrlConn.Close()
			continue
		}

		done := make(chan struct{})
		go sayAlive(ctrlConn, done)

		for {
			req, err := getRequest(ctrlConn)
			if err != nil { // control connction is down
				clog.Error("control connction error: ", err)
				if err == io.EOF {
					ctrlConn.Close()
					close(done)
					break
				}
				continue
			}

			go handleRequest(req, server, tunnelPort)
		}
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
		dbgLog.Fatal("get configuration error: %v\n", err)
	}

	dbgLog = depot.SetDebug(config.Debug)
	dbgLog.Println("depot-local")

	go run(config.ServerAddr, strconv.Itoa(config.ControlPort),
		strconv.Itoa(config.TunnelPort))
	waitSignal()
}
