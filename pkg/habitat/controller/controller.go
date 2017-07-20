// Copyright (c) 2017 Chef Software Inc. and/or applicable contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"
)

type HabitatController struct {
	config Config
	logger log.Logger
}

type Config struct {
	Client *rest.RESTClient
	Scheme *runtime.Scheme
}

func New(config Config, logger log.Logger) HabitatController {
	hc := HabitatController{
		config: config,
		logger: logger,
	}

	return hc
}

// Run starts a Habitat resource controller
func (hc *HabitatController) Run(ctx context.Context) error {
	fmt.Printf("Watching Service Group objects\n")

	_, err := hc.watchCustomResources(ctx)
	if err != nil {
		fmt.Printf("error: Failed to register watch for ServiceGroup resource: %v\n", err)
		return err
	}

	// This channel is closed when the context is canceled or times out.
	<-ctx.Done()

	// Err() contains the error, if any.
	return ctx.Err()
}

func (hc *HabitatController) watchCustomResources(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		hc.config.Client,
		crv1.ServiceGroupResourcePlural,
		apiv1.NamespaceAll,
		fields.Everything())

	_, k8sController := cache.NewInformer(
		source,

		// The object type.
		&crv1.ServiceGroup{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		1*time.Minute,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    hc.onAdd,
			UpdateFunc: hc.onUpdate,
			DeleteFunc: hc.onDelete,
		})

	// The k8sController will start processing events from the API.
	go k8sController.Run(ctx.Done())

	return k8sController, nil
}

func (hc *HabitatController) onAdd(obj interface{}) {
	sg := obj.(*crv1.ServiceGroup)
	fmt.Printf("[CONTROLLER] OnAdd: %s", sg.ObjectMeta.SelfLink)

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use exampleScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj, err := hc.config.Scheme.Copy(sg)
	if err != nil {
		fmt.Printf("ERROR creating a deep copy of ServiceGroup object: %v\n", err)
		return
	}

	sgCopy := copyObj.(*crv1.ServiceGroup)
	sgCopy.Status = crv1.ServiceGroupStatus{
		State:   crv1.ServiceGroupStateProcessed,
		Message: "Successfully processed by controller",
	}

	err = hc.config.Client.Put().
		Name(sg.ObjectMeta.Name).
		Namespace(sg.ObjectMeta.Namespace).
		Resource(crv1.ServiceGroupResourcePlural).
		Body(sgCopy).
		Do().
		Error()

	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED status: %#v\n", sgCopy)
	}
}

func (hc *HabitatController) onUpdate(oldObj, newObj interface{}) {
	oldServiceGroup := oldObj.(*crv1.ServiceGroup)
	newServiceGroup := newObj.(*crv1.ServiceGroup)
	fmt.Printf("[CONTROLLER] OnUpdate oldObj: %s\n", oldServiceGroup.ObjectMeta.SelfLink)
	fmt.Printf("[CONTROLLER] OnUpdate newObj: %s\n", newServiceGroup.ObjectMeta.SelfLink)
}

func (hc *HabitatController) onDelete(obj interface{}) {
	sg := obj.(*crv1.ServiceGroup)
	fmt.Printf("[CONTROLLER] OnDelete %s\n", sg.ObjectMeta.SelfLink)
}
