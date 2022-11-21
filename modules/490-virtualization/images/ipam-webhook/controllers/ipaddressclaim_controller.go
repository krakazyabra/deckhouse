/*
Copyright 2022 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"ipam-webhook/utils"
	"ipam-webhook/webhooks"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	d8v1alpha1 "github.com/deckhouse/deckhouse/modules/490-virtualization/api/v1alpha1"

	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
)

type IPAMValidatorController struct {
	RESTClient rest.Interface
	Logger     kwhlog.Logger
	Webhook    *webhooks.IPAMValidatorWebhook
	IPStore    *utils.IPStore
}

func (c IPAMValidatorController) Start(ctx context.Context) error {
	c.Logger.Infof("starting ipaddressleases controller")

	lw := cache.NewListWatchFromClient(c.RESTClient, "ipaddressleases", v1.NamespaceAll, fields.Everything())
	informer := cache.NewSharedIndexInformer(lw, &d8v1alpha1.VirtualMachineIPAddressLease{}, 12*time.Hour,
		cache.Indexers{
			"namespace_name": func(obj interface{}) ([]string, error) {
				return []string{obj.(*d8v1alpha1.VirtualMachineIPAddressLease).GetName()}, nil
			},
		},
	)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc,
		DeleteFunc: c.deleteFunc,
	})

	stopper := make(chan struct{})
	defer close(stopper)
	defer utilruntime.HandleCrash()
	go informer.Run(stopper)
	c.Logger.Infof("syncronizing")

	//syncronize the cache before starting to process
	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		c.Logger.Errorf("syncronization failed")
		return fmt.Errorf("syncronization failed")
	}
	c.Logger.Infof("syncronization completed")

	c.Logger.Infof("starting validation webhook")
	go c.Webhook.Start()

	<-ctx.Done()
	c.Logger.Infof("shutting down ipaddressleases controller")

	return nil
}

func (c *IPAMValidatorController) addFunc(obj interface{}) {
	claim, ok := obj.(*d8v1alpha1.VirtualMachineIPAddressLease)
	if !ok {
		// object is not VirtualMachineIPAddressLease
		return
	}
	ip := utils.NameToIP(claim.GetName())
	if ip == "" {
		c.Logger.Errorf("failed to allocate %s, wrong name", claim.GetName())
		return
	}
	c.IPStore.Add(ip)
	c.Logger.Infof("allocated %s", claim.GetName())
}

func (c *IPAMValidatorController) deleteFunc(obj interface{}) {
	claim, ok := obj.(*d8v1alpha1.VirtualMachineIPAddressLease)
	if !ok {
		// object is not VirtualMachineIPAddressLease
		return
	}
	ip := utils.NameToIP(claim.GetName())
	if ip == "" {
		c.Logger.Errorf("failed to release %s, wrong name", claim.GetName())
		return
	}
	c.IPStore.Del(ip)
	c.Logger.Infof("released %s", claim.GetName())
}
