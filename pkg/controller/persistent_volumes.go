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
	"github.com/go-kit/kit/log/level"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (hc *HabitatController) cachePersistentVolumes() {
	source := newListWatchFromClientWithLabels(
		hc.config.KubernetesClientset.CoreV1().RESTClient(),
		"persistentvolumes",
		apiv1.NamespaceAll,
		labelListOptions())

	hc.pvInformer = cache.NewSharedIndexInformer(
		source,
		&apiv1.PersistentVolume{},
		resyncPeriod,
		cache.Indexers{},
	)

	hc.pvInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: hc.handlePVDelete,
	})

	hc.pvInformerSynced = hc.pvInformer.HasSynced
}

func (hc *HabitatController) handlePVDelete(obj interface{}) {
	pv, ok := obj.(*apiv1.PersistentVolume)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert PersistentVolumeClaim", "obj", obj)
		return
	}

	// Find out if there's a Habitat object around
	// If so, we have a problem.
	key, err := cache.MetaNamespaceKeyFunc(pv.Spec.ClaimRef)
	if err != nil {
		level.Error(hc.logger).Log("msg", "Failed to get key", "obj", pv)
		return
	}

	_, exists, err := hc.habInformer.GetStore().GetByKey(key)
	if err != nil {
		level.Error(hc.logger).Log("msg", "Failed to get key in Store", "obj", key)
		return
	} else if !exists {
		level.Debug(hc.logger).Log("msg", "No matching Habitat found for PVC", "name", pv.Name)
		return
	} else {
		level.Error(hc.logger).Log("msg", "A PVC has lost its PersistentVolume", "name", pv.Name)
	}
}
