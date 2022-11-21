package main

import (
	"flag"
	"fmt"
	"ipam-webhook/controllers"
	"ipam-webhook/utils"
	"ipam-webhook/webhooks"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	d8v1alpha1 "github.com/deckhouse/deckhouse/modules/490-virtualization/api/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupVersion  = schema.GroupVersion{Group: "deckhouse.io", Version: "v1alpha1"}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
)

type config struct {
	cidrs       cidrFlag
	certFile    string
	keyFile     string
	metricsAddr string
	probeAddr   string
}

var schemee = runtime.NewScheme()

type cidrFlag []string

func (f *cidrFlag) String() string { return "" }
func (f *cidrFlag) Set(s string) error {
	*f = append(*f, s)
	return nil
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(schemee))
	utilruntime.Must(d8v1alpha1.AddToScheme(schemee))
}

func main() {
	logrusLogEntry := logrus.NewEntry(logrus.New())
	logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	logger := kwhlogrus.NewLogrus(logrusLogEntry)
	cfg := &config{}

	flag.Var(&cfg.cidrs, "cidr", "CIDRs enabled to route (multiple flags allowed)")
	flag.StringVar(&cfg.certFile, "tls-cert-file", "", "TLS certificate file")
	flag.StringVar(&cfg.keyFile, "tls-key-file", "", "TLS key file")
	flag.StringVar(&cfg.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&cfg.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	flag.Parse()

	var parsedCIDRs []*net.IPNet
	for _, cidr := range cfg.cidrs {
		_, parsedCIDR, err := net.ParseCIDR(cidr)
		if err != nil || parsedCIDR == nil {
			fmt.Println(err, "failed to parse CIDR")
			os.Exit(1)
		}
		parsedCIDRs = append(parsedCIDRs, parsedCIDR)
	}

	logger.Infof("managed CIDRs: %+v", cfg.cidrs)

	// create webhook
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 schemee,
		MetricsBindAddress:     cfg.metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: cfg.probeAddr,
	})
	if err != nil {
		logger.Errorf("unable to start manager: %s", err)
		os.Exit(1)
	}

	restClient, err := apiutil.RESTClientForGVK(d8v1alpha1.GroupVersion.WithKind("VirtualMachineIPAddressLease"), false, mgr.GetConfig(), serializer.NewCodecFactory(mgr.GetScheme()))
	if err != nil {
		logger.Errorf("unable to create REST client: %s", err)
		os.Exit(1)
	}
	ipStore := utils.NewIPStore()

	controller := controllers.IPAMValidatorController{
		RESTClient: restClient,
		Logger:     logger,
		IPStore:    ipStore,
		Webhook: &webhooks.IPAMValidatorWebhook{
			Logger:   logger,
			CertFile: cfg.certFile,
			KeyFile:  cfg.keyFile,
			CIDRs:    parsedCIDRs,
			IPStore:  ipStore,
		},
	}

	if err := mgr.Add(controller); err != nil {
		logger.Errorf("unable to add ipaddressleases controller to manager %s", err)
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Errorf("unable to set up health check: %s", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Errorf("unable to set up ready check: %s", err)
		os.Exit(1)
	}

	logger.Infof("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Errorf("problem running manager: %s", err)
		os.Exit(1)
	}

}
