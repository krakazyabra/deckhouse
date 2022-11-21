package webhooks

import (
	"context"
	"fmt"
	"ipam-webhook/utils"
	"net"
	"net/http"
	"os"

	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	d8v1alpha1 "github.com/deckhouse/deckhouse/modules/490-virtualization/api/v1alpha1"
)

type IPAMValidatorWebhook struct {
	RESTClient rest.Interface
	CertFile   string
	KeyFile    string
	Logger     kwhlog.Logger
	CIDRs      []*net.IPNet
	IPStore    *utils.IPStore
}

func (v *IPAMValidatorWebhook) Start() {
	// Create our validator
	mcfg := kwhvalidating.WebhookConfig{
		ID:        "ipamValidator",
		Obj:       &d8v1alpha1.VirtualMachineIPAddressLease{},
		Validator: v,
		Logger:    v.Logger,
	}
	wh, err := kwhvalidating.NewWebhook(mcfg)
	if err != nil {
		v.Logger.Errorf("error creating webhook: %s", err)
		os.Exit(1)
	}

	// Get the handler for our webhook.
	whHandler, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: wh, Logger: v.Logger})
	if err != nil {
		v.Logger.Errorf("error creating webhook handler: %s", err)
		os.Exit(1)
	}
	v.Logger.Infof("Listening on :8082")
	err = http.ListenAndServeTLS(":8082", v.CertFile, v.KeyFile, whHandler)
	if err != nil {
		v.Logger.Errorf("error serving webhook: %s", err)
		os.Exit(1)
	}

}

func (v *IPAMValidatorWebhook) Validate(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhvalidating.ValidatorResult, error) {
	if _, ok := obj.(*d8v1alpha1.VirtualMachineIPAddressLease); !ok {
		return nil, fmt.Errorf("not an VirtualMachineIPAddressLease")
	}
	ip := utils.NameToIP(obj.GetName())
	if net.ParseIP(ip) == nil {
		return &kwhvalidating.ValidatorResult{
			Valid:   false,
			Message: "metadata.name does not contain a valid IP address",
		}, nil
	}

	if !v.ipInRange(ip) {
		return &kwhvalidating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("unable to find suitable CIDR for IP address %s, available ranges: %+v", ip, v.CIDRs),
		}, nil
	}
	if v.IPStore.IsAllocated(ip) {
		return &kwhvalidating.ValidatorResult{
			Valid:   false,
			Message: fmt.Sprintf("requested IP address %s is already allocated", ip),
		}, nil
	}
	return &kwhvalidating.ValidatorResult{
		Valid:   true,
		Message: "IP address is valid",
	}, nil
}

func (v *IPAMValidatorWebhook) GetHTTPHandler() (http.Handler, error) {
	var whHandler http.Handler
	mcfg := kwhvalidating.WebhookConfig{
		ID:        "ipamValidator",
		Obj:       &d8v1alpha1.VirtualMachineIPAddressLease{},
		Validator: v,
		Logger:    v.Logger,
	}
	wh, err := kwhvalidating.NewWebhook(mcfg)
	if err != nil {
		return whHandler, fmt.Errorf("error creating webhook: %s", err)
	}

	// Get the handler for our webhook.
	whHandler, err = kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: wh, Logger: v.Logger})
	if err != nil {
		return whHandler, fmt.Errorf("error creating webhook handler: %s", err)
	}
	return whHandler, nil
}

func (v *IPAMValidatorWebhook) ipInRange(ip string) bool {
	for _, cidr := range v.CIDRs {
		if cidr.Contains(net.ParseIP(ip)) {
			return true
		}
	}
	return false
}
