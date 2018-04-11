// Copyright (c) 2018 Chef Software Inc. and/or applicable contributors
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

package v1beta2

import (
	"fmt"

	habv1beta1 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta1"

	"github.com/go-kit/kit/log/level"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const persistentVolumeName = "persistent"

func (hc *HabitatController) newStatefulSet(h *habv1beta1.Habitat) (*appsv1beta1.StatefulSet, error) {
	// This value needs to be passed as a *int32, so we convert it, assign it to a
	// variable and afterwards pass a pointer to it.
	count := int32(h.Spec.Count)

	// Set the service arguments we send to Habitat.
	var habArgs []string
	if h.Spec.Service.Group != "" {
		// When a service is started without explicitly naming the group,
		// it's assigned to the default group.
		habArgs = append(habArgs,
			"--group", h.Spec.Service.Group)
	}

	// As we want to label our pods with the
	// topology type we set standalone as the default one.
	// We do not need to pass this to habitat, as if no topology
	// is set, habitat by default sets standalone topology.
	topology := habv1beta1.TopologyStandalone

	if h.Spec.Service.Topology == habv1beta1.TopologyLeader {
		topology = habv1beta1.TopologyLeader
	}

	path := fmt.Sprintf("%s/%s", configMapDir, peerFilename)

	habArgs = append(habArgs,
		"--topology", topology.String(),
		"--peer-watch-file", path,
	)

	// Runtime binding.
	// One Service connects to another forming a producer/consumer relationship.
	for _, bind := range h.Spec.Service.Bind {
		// Pass --bind flag.
		bindArg := fmt.Sprintf("%s:%s.%s", bind.Name, bind.Service, bind.Group)
		habArgs = append(habArgs,
			"--bind", bindArg)
	}

	base := &appsv1beta1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.Name,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: habv1beta1.SchemeGroupVersion.String(),
					Kind:       habv1beta1.HabitatKind,
					Name:       h.Name,
					UID:        h.UID,
				},
			},
		},
		Spec: appsv1beta1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					habv1beta1.HabitatNameLabel: h.Name,
				},
			},
			Replicas:            &count,
			PodManagementPolicy: appsv1beta1.ParallelPodManagement,
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						habv1beta1.HabitatLabel:     "true",
						habv1beta1.HabitatNameLabel: h.Name,
						habv1beta1.TopologyLabel:    topology.String(),
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "habitat-service",
							Image: h.Spec.Image,
							Args:  habArgs,
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "config",
									MountPath: configMapDir,
									ReadOnly:  true,
								},
							},
							Env: h.Spec.Env,
						},
					},
					// Define the volume for the ConfigMap.
					Volumes: []apiv1.Volume{
						{
							Name: "config",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: configMapName,
									},
									Items: []apiv1.KeyToPath{
										{
											Key:  peerFile,
											Path: peerFilename,
										},
									},
								},
							},
						},
					},
				},
			},
			UpdateStrategy: appsv1beta1.StatefulSetUpdateStrategy{
				Type: appsv1beta1.StatefulSetUpdateStrategyType(appsv1beta1.RollingUpdateDeploymentStrategyType),
			},
		},
	}

	spec := &base.Spec
	tSpec := &spec.Template.Spec

	// If we have a secret name present we should mount that secret.
	if h.Spec.Service.ConfigSecretName != "" {
		// Let's make sure our secret is there before mounting it.
		secret, err := hc.config.KubernetesClientset.CoreV1().Secrets(h.Namespace).Get(h.Spec.Service.ConfigSecretName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		secretVolume := &apiv1.Volume{
			Name: userConfigFilename,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: secret.Name,
					Items: []apiv1.KeyToPath{
						{
							Key:  userTOMLFile,
							Path: userTOMLFile,
						},
					},
				},
			},
		}

		secretVolumeMount := &apiv1.VolumeMount{
			Name: userConfigFilename,
			// The Habitat supervisor creates a directory for each service under /hab/svc/<servicename>.
			// We need to place the user.toml file in there in order for it to be detected.
			MountPath: fmt.Sprintf("/hab/user/%s/config", h.Spec.Service.Name),
			ReadOnly:  false,
		}

		tSpec.Containers[0].VolumeMounts = append(tSpec.Containers[0].VolumeMounts, *secretVolumeMount)
		tSpec.Volumes = append(tSpec.Volumes, *secretVolume)
	}

	// Mount Persistent Volume, if requested.
	if ps := h.Spec.PersistentStorage; ps != nil {
		vm := &apiv1.VolumeMount{
			Name:      persistentVolumeName,
			MountPath: ps.MountPath,
		}

		tSpec.Containers[0].VolumeMounts = append(tSpec.Containers[0].VolumeMounts, *vm)

		q, err := resource.ParseQuantity(ps.Size)
		if err != nil {
			return nil, fmt.Errorf("Could not parse PersistentStorage.Size: %v", err)
		}

		spec.VolumeClaimTemplates = []apiv1.PersistentVolumeClaim{
			apiv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      persistentVolumeName,
					Namespace: h.Namespace,
					Labels: map[string]string{
						habv1beta1.HabitatLabel:     "true",
						habv1beta1.HabitatNameLabel: h.Name,
					},
				},
				Spec: apiv1.PersistentVolumeClaimSpec{
					AccessModes: []apiv1.PersistentVolumeAccessMode{
						apiv1.ReadWriteOnce,
					},
					StorageClassName: &ps.StorageClassName,
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{
							apiv1.ResourceStorage: q,
						},
					},
				},
			},
		}
	}

	// Handle ring key, if one is specified.
	if ringSecretName := h.Spec.Service.RingSecretName; ringSecretName != "" {
		s, err := hc.config.KubernetesClientset.CoreV1().Secrets(apiv1.NamespaceDefault).Get(ringSecretName, metav1.GetOptions{})
		if err != nil {
			level.Error(hc.logger).Log("msg", "Could not find Secret containing ring key")
			return nil, err
		}

		// The filename under which the ring key is saved.
		ringKeyFile := fmt.Sprintf("%s.%s", ringSecretName, ringKeyFileExt)

		// Extract the bare ring name, by removing the revision.
		// Validation has already been performed by this point.
		ringName := ringRegexp.FindStringSubmatch(ringSecretName)[1]

		v := &apiv1.Volume{
			Name: ringSecretName,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: s.Name,
					Items: []apiv1.KeyToPath{
						{
							Key:  ringSecretKey,
							Path: ringKeyFile,
						},
					},
				},
			},
		}

		vm := &apiv1.VolumeMount{
			Name:      ringSecretName,
			MountPath: "/hab/cache/keys",
			// This directory cannot be made read-only, as the supervisor writes to
			// it during its operation.
			ReadOnly: false,
		}

		// Mount ring key file.
		tSpec.Volumes = append(tSpec.Volumes, *v)
		tSpec.Containers[0].VolumeMounts = append(tSpec.Containers[0].VolumeMounts, *vm)

		// Add --ring argument to supervisor invocation.
		tSpec.Containers[0].Args = append(tSpec.Containers[0].Args, "--ring", ringName)
	}

	return base, nil
}

func (hc *HabitatController) cacheStatefulSets() {
	hc.stsInformer = hc.config.KubeInformerFactory.Apps().V1beta1().StatefulSets().Informer()

	hc.stsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    hc.handleStsAdd,
		UpdateFunc: hc.handleStsUpdate,
		DeleteFunc: hc.handleStsDelete,
	})

	hc.stsInformerSynced = hc.stsInformer.HasSynced
}

func (hc *HabitatController) handleStsAdd(obj interface{}) {
	sts, ok := obj.(*appsv1beta1.StatefulSet)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert StatefulSet", "obj", obj)
		return
	}

	h, err := hc.getHabitatFromLabeledResource(sts)
	if err != nil {
		level.Error(hc.logger).Log("msg", "Could not find Habitat for StatefulSet", "name", sts.Name)
		return
	}

	hc.enqueue(h)
}

func (hc *HabitatController) handleStsUpdate(oldObj, newObj interface{}) {
	sts, ok := newObj.(*appsv1beta1.StatefulSet)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert StatefulSet", "obj", newObj)
		return
	}

	h, err := hc.getHabitatFromLabeledResource(sts)
	if err != nil {
		level.Error(hc.logger).Log("msg", "Could not find Habitat for StatefulSet", "name", sts.Name)
		return
	}

	hc.enqueue(h)
}

func (hc *HabitatController) handleStsDelete(obj interface{}) {
	sts, ok := obj.(*appsv1beta1.StatefulSet)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert StatefulSet", "obj", obj)
		return
	}

	h, err := hc.getHabitatFromLabeledResource(sts)
	if err != nil {
		// Could not find Habitat, it must have already been removed.
		level.Debug(hc.logger).Log("msg", "Could not find Habitat for StatefulSet", "name", sts.Name)
		return
	}

	hc.enqueue(h)
}
