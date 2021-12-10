package main

import (
	"os"

	apiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"

	"github.com/Congrool/nodes-grouping/cmd/controller-manager/app"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	ctx := apiserver.SetupSignalContext()

	if err := app.NewControllerManagerCommand(ctx).Execute(); err != nil {
		os.Exit(1)
	}
}
