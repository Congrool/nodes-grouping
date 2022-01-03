package main

import (
	"context"

	groupmanagementv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/schedulerextender"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme = runtime.NewScheme()
	// setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(groupmanagementv1alpha1.AddToScheme(scheme))
	utilruntime.Must(groupmanagementv1alpha1.AddToScheme(scheme))
	klog.InitFlags(nil)
}

func main() {

	// opts := zap.Options{}
	// opts.BindFlags(flag.CommandLine)
	// flag.Parse()
	// ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	config := ctrl.GetConfigOrDie()
	defer klog.Flush()

	client, err := client.New(config, client.Options{
		Scheme: scheme,
	})

	if err != nil {
		klog.Fatalf("failed to get client, %v", err)
	}

	server := schedulerextender.NewPolicyServer(context.Background(), client)
	server.Run()
}
