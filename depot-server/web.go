package main

import (
	"html/template"
	"net"
	"net/http"
	"path/filepath"

	"github.com/choueric/clog"
	"github.com/choueric/depot"
)

type webInfoT struct {
	Version    string
	CtrlAddr   string
	TunnelHost string
	dir        string
}

var webInfo webInfoT

func initWebInfo() {
	webInfo.Version = depot.VERSION
	webInfo.dir = depot.GetDefaultConfigDir()
}

func updateWebInfo() {
	if ctrlInfo.ctrlConn != nil {
		webInfo.CtrlAddr = ctrlInfo.ctrlConn.RemoteAddr().String()
	} else {
		webInfo.CtrlAddr = ""
	}
	dbgLog.Println("CtrlAddr", webInfo.CtrlAddr)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	updateWebInfo()

	file := filepath.Join(webInfo.dir, "root.html")
	t, err := template.ParseFiles(file)
	if err != nil {
		clog.Fatal(err)
	}
	t.Execute(w, &webInfo)
}

func serveWeb(host, webPort string) {
	initWebInfo()

	http.Handle("/js/", http.FileServer(http.Dir(webInfo.dir)))
	http.Handle("/css/", http.FileServer(http.Dir(webInfo.dir)))

	http.HandleFunc("/", rootHandler)

	addr := net.JoinHostPort(host, webPort)
	dbgLog.Printf("start listen web at %v ...\n", addr)
	http.ListenAndServe(addr, nil)
}
