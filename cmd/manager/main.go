package main

import (
	"context"
	"flag"
	"fmt"

	"log"
	"os"
	"runtime"

	osv1 "github.com/openshift/api/operator/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	operatorSdkMetrics "github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/controller"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/monitoring"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/util/k8s"
)

var (
	operatorSdkMetricsHost           = "0.0.0.0"
	operatorSdkHttpMetricsPort int32 = 8383
	operatorSdkCrMetricsPort   int32 = 8686
)

func printVersion() {
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Printf("version of operator-sdk: %v", sdkVersion.Version)
	log.Printf("version of cluster-network-addons-operator: %v", os.Getenv("OPERATOR_VERSION"))
}

func main() {
	// Add flags registered by imported packages (e.g. controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Printf("failed to get watch namespace: %v", err)
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Printf("failed to get apiserver config: %v", err)
		os.Exit(1)
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, "cluster-network-addons-operator-lock")
	if err != nil {
		log.Printf("failed to become operator leader: %v", err)
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", operatorSdkMetricsHost, operatorSdkHttpMetricsPort),
		MapperProvider:     k8s.NewDynamicRESTMapper,
	})
	if err != nil {
		log.Printf("failed to instantiate new operator manager: %v", err)
		os.Exit(1)
	}

	log.Print("registering Components")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Printf("failed adding network addons scheme to the client: %v", err)
		os.Exit(1)
	}
	if err := osv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Printf("failed adding openshift scheme to the client: %v", err)
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Printf("failed setting up operator controllers: %v", err)
		os.Exit(1)
	}

	if err := monitoring.StartPrometheus(); err != nil {
		log.Printf("failed starting Prometheus endpoint: %v", err)
		os.Exit(1)
	}

	servicePorts := []v1.ServicePort{
		{Port: operatorSdkHttpMetricsPort, Name: operatorSdkMetrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorSdkHttpMetricsPort}},
		{Port: operatorSdkCrMetricsPort, Name: operatorSdkMetrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorSdkCrMetricsPort}},
	}

	// Create Service object to expose the metrics port.
	if _, err = operatorSdkMetrics.CreateMetricsService(ctx, cfg, servicePorts); err != nil {
		log.Printf("failed to create metrics server: %v", err)
		os.Exit(1)
	}

	log.Print("starting the operator manager")

	// Start the operator manager
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Printf("manager exited with non-zero: %v", err)
		os.Exit(1)
	}
}
