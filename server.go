package jwa

import (
	"net/http"
)

// 调用 ListenAndServe 后设置此字段为true
// 程序内一些函数在启动后不能再调用 这里用来检查
var serverRunning bool

func ListenAndServeWithServer(server *http.Server) error {

	start()
	server.Handler = ServeMux

	return server.ListenAndServe()
}

func ListenAndServeTLSWithServer(server *http.Server, certFile, keyFile string) error {

	start()
	server.Handler = ServeMux

	return server.ListenAndServeTLS(certFile, keyFile)
}

func start() {
	serverRunning = true
	registerHandler()
}
