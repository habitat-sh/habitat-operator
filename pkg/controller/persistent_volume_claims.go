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
	"fmt"

	"github.com/go-kit/kit/log/level"
	habv1beta1 "github.com/kinvolk/habitat-operator/pkg/apis/habitat/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (hc *HabitatController) cachePersistentVolumeClaims() {
	source := newListWatchFromClientWithLabels(
		hc.config.KubernetesClientset.CoreV1().RESTClient(),
		"persistentvolumeclaims",
		apiv1.NamespaceAll,
		labelListOptions())

	hc.cmInformer = cache.NewSharedIndexInformer(
		source,
		&apiv1.PersistentVolumeClaim{},
		resyncPeriod,
		cache.Indexers{},
	)

	hc.cmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: hc.handlePVCUpdate,
		DeleteFunc: hc.handlePVCDelete,
	})

	hc.cmInformerSynced = hc.pvcInformer.HasSynced
}

// findHabForPVC looks for a matching Habitat object for a PVC.
// This method is called when the PVC has been deleted or its status is Lost,
// so finding a matching Habitat means that this object has lost its storage.
func (hc *HabitatController) findHabForPVC(pvc *apiv1.PersistentVolumeClaim) {
	// TODO(asymmetric) Is there a way to find the Habitat without having to produce the key ourselves?
	key := fmt.Sprintf("%s/%s", pvc.Namespace, pvc.Labels[habv1beta1.HabitatNameLabel])

	_, exists, err := hc.habInformer.GetStore().GetByKey(key)
	if err != nil {
		level.Error(hc.logger).Log("msg", "Failed to get key in Store", "obj", key)
		return
	} else if !exists {
		level.Debug(hc.logger).Log("msg", "No matching Habitat found for PVC", "name", pvc.Name)
		return
	} else {
		level.Error(hc.logger).Log("msg", "A PVC has lost its PersistentVolume", "name", pvc.Name)
	}
}

func (hc *HabitatController) handlePVCUpdate(oldObj, newObj interface{}) {
	pvc, ok := newObj.(*apiv1.PersistentVolumeClaim)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert PersistentVolumeClaim", "obj", newObj)
		return
	}

	if pvc.Status.Phase == apiv1.ClaimLost {
		hc.findHabForPVC(pvc)
	}
}

func (hc *HabitatController) handlePVCDelete(obj interface{}) {
	pvc, ok := obj.(*apiv1.PersistentVolumeClaim)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert PersistentVolumeClaim", "obj", obj)
		return
	}

	hc.findHabForPVC(pvc)
}
