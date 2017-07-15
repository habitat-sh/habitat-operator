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

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"
)

type HabitatController struct {
	HabitatClient *rest.RESTClient
	HabitatScheme *runtime.Scheme
}

// Run starts a Habitat resource controller
func (c *HabitatController) Run(ctx context.Context) error {
	fmt.Printf("Watching Service Group objects\n")

	_, err := c.watchServiceGroups(ctx)
	if err != nil {
		fmt.Printf("error: Failed to register watch for ServiceGroup resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *HabitatController) watchServiceGroups(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.HabitatClient,
		crv1.ServiceGroupResourcePlural,
		apiv1.NamespaceAll,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,

		// The object type.
		&crv1.ServiceGroup{},

		// resyncPeriod
		// Every resyncPeriod, all resources in the cache will retrigger events.
		// Set to 0 to disable the resync.
		1*time.Minute,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})

	go controller.Run(ctx.Done())

	return controller, nil
}

func (c *HabitatController) onAdd(obj interface{}) {
	sg := obj.(*crv1.ServiceGroup)
	fmt.Printf("[CONTROLLER] OnAdd: %s", sg.ObjectMeta.SelfLink)

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use exampleScheme.Copy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	copyObj, err := c.HabitatScheme.Copy(sg)
	if err != nil {
		fmt.Printf("ERROR creating a deep copy of ServiceGroup object: %v\n", err)
		return
	}

	sgCopy := copyObj.(*crv1.ServiceGroup)
	sgCopy.Status = crv1.ServiceGroupStatus{
		State:   crv1.ServiceGroupStateProcessed,
		Message: "Successfully processed by controller",
	}

	err = c.HabitatClient.Put().
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

func (c *HabitatController) onUpdate(oldObj, newObj interface{}) {
	oldServiceGroup := oldObj.(*crv1.ServiceGroup)
	newServiceGroup := newObj.(*crv1.ServiceGroup)
	fmt.Printf("[CONTROLLER] OnUpdate oldObj: %s\n", oldServiceGroup.ObjectMeta.SelfLink)
	fmt.Printf("[CONTROLLER] OnUpdate newObj: %s\n", newServiceGroup.ObjectMeta.SelfLink)
}

func (c *HabitatController) onDelete(obj interface{}) {
	sg := obj.(*crv1.ServiceGroup)
	fmt.Printf("[CONTROLLER] OnDelete %s\n", sg.ObjectMeta.SelfLink)
}
