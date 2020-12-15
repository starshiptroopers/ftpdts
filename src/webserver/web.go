package webserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Opts is a ftpdt options
type Opts struct {
	Port        uint
	Host        string
	DataStorage DataStorage //data storage
	Logger      *log.Logger //Where log will be written to (default to stdout)
}

type DataStorage interface {
	Get(uid string) (payload interface{}, createdAt time.Time, err error)
	Put(uid string, payload interface{}, ttl *time.Duration) error
}

type WebServer struct {
	logger *log.Logger
	ds     DataStorage
	port   uint
	server *http.Server
}

func New(o Opts) *WebServer {
	var mux http.ServeMux

	s := &WebServer{
		o.Logger,
		o.DataStorage,
		o.Port,
		&http.Server{
			Addr:    fmt.Sprintf("%s:%d", o.Host, o.Port),
			Handler: &mux,
		},
	}
	mux.HandleFunc("/reg", s.reg)

	return s
}

func (s *WebServer) Run() error {
	s.logger.Printf("Starting the web server at %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *WebServer) reg(res http.ResponseWriter, req *http.Request) {
	s.logger.Printf("New reg request")
	_, _ = fmt.Fprint(res, "The TEST")
}

func (s *WebServer) Shutdown() {
	ctx := context.Background()
	s.logger.Printf("Shutting down the web server")
	_ = s.server.Shutdown(ctx)
}
