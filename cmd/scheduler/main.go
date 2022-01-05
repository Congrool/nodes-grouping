package main

import (
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	_ "github.com/Congrool/nodes-grouping/pkg/apis/config/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/scheduler"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewSchedulerCommand(
		app.WithPlugin(scheduler.Name, scheduler.New),
	)

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
