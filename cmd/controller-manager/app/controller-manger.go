package app

import (
	"context"
	"net"
	"strconv"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/Congrool/nodes-grouping/cmd/controller-manager/app/options"
	groupv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/group/v1alpha1"
	policyv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/policy/v1alpha1"
	groupcontroller "github.com/Congrool/nodes-grouping/pkg/controllers/group"
	policycontroller "github.com/Congrool/nodes-grouping/pkg/controllers/policy"
)

// aggregatedScheme aggregates Kubernetes and extended schemems.
var aggregatedScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(scheme.AddToScheme(aggregatedScheme))
	utilruntime.Must(groupv1alpha1.AddToScheme(aggregatedScheme))
	utilruntime.Must(policyv1alpha1.AddToScheme(aggregatedScheme))
}

// NewControllerManagerCommand creates a *cobra.Command object with default parameters
func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "node-group-controller-manager",
		Long: `The node group controller manager run a bunch of controllers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO:
			// need validation of opts
			return Run(ctx, opts)
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

// Run runs the controller-manager with options. This should never exit.
func Run(ctx context.Context, opts *options.Options) error {
	klog.Infoln("execute Run")
	config, err := controllerruntime.GetConfig()
	if err != nil {
		panic(err)
	}
	config.QPS, config.Burst = opts.KubeAPIQPS, opts.KubeAPIBurst
	controllerManager, err := controllerruntime.NewManager(config, controllerruntime.Options{
		Scheme:                     aggregatedScheme,
		LeaderElection:             opts.LeaderElection.LeaderElect,
		LeaderElectionID:           "80807133.group.kubeedge.io",
		LeaderElectionNamespace:    opts.LeaderElection.ResourceNamespace,
		LeaderElectionResourceLock: opts.LeaderElection.ResourceLock,
		HealthProbeBindAddress:     net.JoinHostPort(opts.BindAddress, strconv.Itoa(opts.SecurePort)),
		LivenessEndpointName:       "/healthz",
	})
	if err != nil {
		klog.Errorf("failed to build controller manager: %v", err)
		return err
	}

	klog.Infoln("execute Healthz")
	if err := controllerManager.AddHealthzCheck("ping", healthz.Ping); err != nil {
		klog.Errorf("failed to add health check endpoint: %v", err)
		return err
	}

	klog.Infoln("execute Controllers")
	setupControllers(controllerManager, opts, ctx.Done())

	klog.Infoln("execute Start")
	// blocks until the context is done
	if err := controllerManager.Start(ctx); err != nil {
		klog.Errorf("controller manager exits unexpectedly: %v", err)
		return err
	}

	// never reach here
	return nil
}

// setupControllers initialize coontrollers and setup one by one.
func setupControllers(mgr controllerruntime.Manager, opts *options.Options, stopChan <-chan struct{}) {
	// restConfig := mgr.GetConfig()
	// dynamicClientSet := dynamic.NewForConfigOrDie(restConfig)
	// discoverClientSet := discovery.NewDiscoveryClientForConfigOrDie(restConfig)

	// controlPlaneInformerManager := informermanager.NewSingleClusterInformerManager(dynamicClientSet, 0, stopChan)

	nodeGroupController := &groupcontroller.Controller{
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor(groupcontroller.ControllerName),
	}

	propagationPolicyController := &policycontroller.Controller{
		Client: mgr.GetClient(),
	}

	klog.Infoln("setup nodegroup controller")
	if err := nodeGroupController.SetupWithManager(mgr); err != nil {
		klog.Errorf("Failed to setup nodegroup controller: %v", err)
	}

	klog.Infoln("setup propagationpolicy controller")
	if err := propagationPolicyController.SetupWithManager(mgr); err != nil {
		klog.Errorf("Failed to setup propogation policy controller: %v", err)
	}
}
