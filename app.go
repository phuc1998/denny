package denny

import (
	"github.com/gin-gonic/gin"
	"github.com/whatvn/denny/log"
	"sync"
)

type Context = gin.Context

type HandleFunc = gin.HandlerFunc

type methodHandlerMap struct {
	method  HttpMethod
	handler HandleFunc
	*log.Log
}

type denny struct {
	sync.Mutex
	handlerMap map[string]*methodHandlerMap
	*gin.Engine
}

func NewServer() *denny {
	return &denny{
		handlerMap: make(map[string]*methodHandlerMap),
		Engine:     gin.New(),
	}
}

func (r *denny) Controller(path string, method HttpMethod, ctl controller) {
	r.Lock()
	defer r.Unlock()
	ctl.init()
	m := &methodHandlerMap{
		method:  method,
		handler: ctl.Handle,
		Log:     log.New(path),
	}
	m.Infof("setting up handler for path %s, method %V", path, method)
	r.handlerMap[path] = m
}

func (r *denny) initRoute() {
	for p, m := range r.handlerMap {
		switch m.method {
		case HttpGet:
			r.GET(p, m.handler)
		case HttpPost:
			r.POST(p, m.handler)
		case HttpDelete:
			r.DELETE(p, m.handler)
		case HttpOption:
			r.OPTIONS(p, m.handler)
		case HttpPatch:
			r.PATCH(p, m.handler)
		}
	}
}

func (r *denny) WithMiddleware(middleware ...HandleFunc) {
	r.Use(middleware...)
}

func (r *denny) Start(addrs ...string) error {
	r.initRoute()
	return r.Run(addrs...)
}