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
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

type Server interface {
	Run()
}

type server struct {
	httpserver        *http.Server
	controllerManager controllerruntime.Manager
	scheduler         scheduler.SchedulerExtender
	ctx               context.Context
}

func NewPolicyServer(ctx context.Context, controllerManager controllerruntime.Manager) Server {
	s := &server{
		httpserver: &http.Server{
			Addr: fmt.Sprintf("%s:%s", constants.ServerListeningAddr, constants.ServerListeningPort),
		},
		controllerManager: controllerManager,
		ctx:               ctx,
	}
	s.scheduler = scheduler.NewSchedulerExtender(ctx, controllerManager.GetClient())

	mux := mux.NewRouter()
	s.registerHandler(mux)
	s.httpserver.Handler = mux

	return s
}

func (s *server) Run() {
	klog.Info("starting scheduler extender server, waiting for cache sync")
	if !s.controllerManager.GetCache().WaitForCacheSync(s.ctx) {
		panic("failed on WaitForCacheSync")
	}
	go func() {
		err := s.httpserver.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	<-s.ctx.Done()
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
