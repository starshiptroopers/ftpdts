package webserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Opts is a ftpdt options
type Opts struct {
	Port        uint
	DataStorage DataStorage //data storage
	LogWriter   io.Writer   //Where log will be written to (default to stdout)
}

type DataStorage interface {
	Get(uid string) (payload interface{}, createdAt time.Time, err error)
	Put(uid string, payload interface{}, ttl *time.Duration) error
}

type WebServer struct {
	logWriter io.Writer
	ds        DataStorage
	port      uint
	server    *http.Server
}

func New(o Opts) *WebServer {
	var mux http.ServeMux

	s := &WebServer{
		o.LogWriter,
		o.DataStorage,
		o.Port,
		&http.Server{
			Addr:    fmt.Sprintf(":%d", o.Port),
			Handler: &mux,
		},
	}
	mux.HandleFunc("/reg", s.reg)

	return s
}

func (s *WebServer) Run() error {
	return s.server.ListenAndServe()
}

func (s *WebServer) reg(res http.ResponseWriter, req *http.Request) {
	_, _ = fmt.Fprint(res, "The TEST")
}

func (s *WebServer) Shutdown() {
	ctx := context.Background()
	_ = s.server.Shutdown(ctx)
}
