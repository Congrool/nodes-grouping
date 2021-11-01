package server

// Accept http request from scheduler to filter and score nodes
// according to the policy recorded at pod's label.

import (
	"context"
	"net/http"

	"github.com/Congrool/nodes-grouping/pkg/server/scheduler"
	"github.com/Congrool/nodes-grouping/pkg/utils"
	"github.com/gorilla/mux"
)

type Server interface {
	Run()
}

type server struct {
	httpserver *http.Server
	// TODO: Informer
	scheduler scheduler.SchedulerExtender
	ctx       context.Context
}

func NewPolicyServer(ctx context.Context) {
	// TODO:
	s := &server{
		httpserver: &http.Server{
			Addr: "0.0.0.0:10053",
		},
		ctx: ctx,
	}

	mux := mux.NewRouter()
	s.registerHandler(mux)
	s.httpserver.Handler = mux
}

func (s *server) Run(stopCh <-chan struct{}) {
	go func() {
		err := s.httpserver.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	<-stopCh
	s.httpserver.Shutdown(s.ctx)
}

func (s *server) registerHandler(mux *mux.Router) {
	mux.Handle("/schedule/filter", s.buildFilterHandler())
	// mux.HandleFunc("/schedule/prioritize", s.prioritize)
	// mux.HandleFunc("/schedule/bind", s.bind)
	mux.Methods("POST")
}

func (s *server) buildFilterHandler() http.Handler {
	handler := scheduler.WithFilterHandler(s.scheduler.Filter())
	handler = utils.WithCheck(handler)
	return handler
}
