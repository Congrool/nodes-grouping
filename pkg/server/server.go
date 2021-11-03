package server

// Accept http request from scheduler to filter and score nodes
// according to the policy recorded at pod's label.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Congrool/nodes-grouping/pkg/server/constants"
	"github.com/Congrool/nodes-grouping/pkg/server/scheduler"
	"github.com/Congrool/nodes-grouping/pkg/utils"
	"github.com/gorilla/mux"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Server interface {
	Run()
}

type server struct {
	httpserver *http.Server
	scheduler  scheduler.SchedulerExtender
	ctx        context.Context
}

func NewPolicyServer(ctx context.Context, client client.Client) {
	s := &server{
		httpserver: &http.Server{
			Addr: fmt.Sprintf("%s:%s", constants.ServerListeningAddr, constants.ServerListeningPort),
		},
		ctx: ctx,
	}
	s.scheduler = scheduler.NewSchedulerExtender(ctx, client)

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
	mux.Handle("/schedule/prioritize", s.buildPrioritizeHandler())
	mux.Methods("POST")
}

func (s *server) buildFilterHandler() http.Handler {
	handler := scheduler.WithFilterHandler(s.scheduler.Filter)
	handler = utils.WithCheck(handler)
	return handler
}

func (s *server) buildPrioritizeHandler() http.Handler {
	handler := scheduler.WithPrioritizeHander(s.scheduler.Prioritize)
	handler = utils.WithCheck(handler)
	return handler
}
