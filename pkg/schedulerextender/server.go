package schedulerextender

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Congrool/nodes-grouping/pkg/schedulerextender/constants"
	"github.com/Congrool/nodes-grouping/pkg/schedulerextender/extender"
	"github.com/Congrool/nodes-grouping/pkg/schedulerextender/extender/filter"
	"github.com/Congrool/nodes-grouping/pkg/schedulerextender/extender/prioritizer"
	"github.com/Congrool/nodes-grouping/pkg/utils"
	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Server interface {
	Run()
}

type server struct {
	httpserver *http.Server
	scheduler  extender.SchedulerExtender
	ctx        context.Context
}

func NewPolicyServer(ctx context.Context, client client.Client) Server {
	s := &server{
		httpserver: &http.Server{
			Addr: fmt.Sprintf("%s:%s", constants.ServerListeningAddr, constants.ServerListeningPort),
		},
		ctx: ctx,
	}
	s.scheduler = extender.NewSchedulerExtender(ctx, client)

	mux := mux.NewRouter()
	s.registerHandler(mux)
	s.httpserver.Handler = mux

	return s
}

func (s *server) Run() {
	klog.Info("starting scheduler extender server")
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
	handler := filter.WithFilterHandler(s.scheduler.Filter)
	handler = utils.WithCheck(handler)
	return handler
}

func (s *server) buildPrioritizeHandler() http.Handler {
	handler := prioritizer.WithPrioritizeHander(s.scheduler.Prioritize)
	handler = utils.WithCheck(handler)
	return handler
}
