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
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"
)

const (
	resyncPeriod = 1 * time.Minute
	peerFile     = "peer-watch-file"
)

type HabitatController struct {
	config Config
	logger log.Logger
}

type Config struct {
	HabitatClient       *rest.RESTClient
	KubernetesClientset *kubernetes.Clientset
	Scheme              *runtime.Scheme
}

func New(config Config, logger log.Logger) (*HabitatController, error) {
	if config.HabitatClient == nil {
		return nil, errors.New("invalid controller config: no HabitatClient")
	}
	if config.KubernetesClientset == nil {
		return nil, errors.New("invalid controller config: no KubernetesClientset")
	}
	if config.Scheme == nil {
		return nil, errors.New("invalid controller config: no Schema")
	}
	if logger == nil {
		return nil, errors.New("invalid controller config: no logger")
	}

	hc := &HabitatController{
		config: config,
		logger: logger,
	}

	return hc, nil
}

// Run starts a Habitat resource controller.
func (hc *HabitatController) Run(ctx context.Context) error {
	level.Info(hc.logger).Log("msg", "Watching Service Group objects")

	hc.watchCustomResources(ctx)

	hc.watchPods(ctx)

	// This channel is closed when the context is canceled or times out.
	<-ctx.Done()

	// Err() contains the error, if any.
	return ctx.Err()
}

func (hc *HabitatController) watchCustomResources(ctx context.Context) {
	source := cache.NewListWatchFromClient(
		hc.config.HabitatClient,
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
		resyncPeriod,

		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			AddFunc:    hc.onAdd,
			UpdateFunc: hc.onUpdate,
			DeleteFunc: hc.onDelete,
		})

	// The k8sController will start processing events from the API.
	go k8sController.Run(ctx.Done())
}

func (hc *HabitatController) onAdd(obj interface{}) {
	sg, ok := obj.(*crv1.ServiceGroup)
	if !ok {
		level.Error(hc.logger).Log("msg", "unknown event type")
		return
	}

	level.Debug(hc.logger).Log("function", "onAdd", "msg", sg.ObjectMeta.SelfLink)

	// Validate object.
	if err := validateCustomObject(*sg); err != nil {
		if vErr, ok := err.(validationError); ok {
			level.Error(hc.logger).Log("type", "validation error", "msg", err, "key", vErr.Key)
			return
		}

		level.Error(hc.logger).Log("msg", err)
		return
	}

	level.Debug(hc.logger).Log("msg", "validated object")

	group := "default"
	if sg.Spec.Habitat.Group != "" {
		group = sg.Spec.Habitat.Group
	}

	// This value needs to be passed as a *int32, so we convert it, assign it to a
	// variable and afterwards pass a pointer to it.
	count := int32(sg.Spec.Count)

	// Create a deployment.
	deployment := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: sg.Name,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Replicas: &count,
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"habitat":       "true",
						"service-group": sg.Name,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "habitat-service",
							Image: sg.Spec.Image,
							Args: []string{
								"--group", group,
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/habitat-operator",
									ReadOnly:  true,
								},
							},
						},
					},
					// Define the volume for the ConfigMap.
					Volumes: []apiv1.Volume{
						{
							Name: "config",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: configMapName(sg.Name),
									},
									Items: []apiv1.KeyToPath{
										{
											Key:  peerFile,
											Path: "peer-ip",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	d, err := hc.config.KubernetesClientset.AppsV1beta1Client.Deployments(apiv1.NamespaceDefault).Create(deployment)
	if err != nil {
		level.Error(hc.logger).Log("msg", err)
		return
	}

	level.Info(hc.logger).Log("msg", "created deployment", "name", d.GetObjectMeta().GetName())

	// Create the ConfigMap for the peer watch file.
	configMap := newConfigMap(sg.Name, d.UID, "")
	_, err = hc.config.KubernetesClientset.CoreV1Client.ConfigMaps(apiv1.NamespaceDefault).Create(configMap)
	if err != nil {
		level.Error(hc.logger).Log("msg", err)
		return
	}

	level.Debug(hc.logger).Log("msg", "created ConfigMap with peer IP", "object", configMap.Data["peer-ip"])
}

func (hc *HabitatController) onUpdate(oldObj, newObj interface{}) {
	oldServiceGroup := oldObj.(*crv1.ServiceGroup)
	newServiceGroup := newObj.(*crv1.ServiceGroup)
	level.Info(hc.logger).Log("function", "onUpdate", "msg", fmt.Sprintf("oldObj: %s, newObj: %s", oldServiceGroup.ObjectMeta.SelfLink, newServiceGroup.ObjectMeta.SelfLink))
}

func (hc *HabitatController) onDelete(obj interface{}) {
	sg, ok := obj.(*crv1.ServiceGroup)
	if !ok {
		level.Error(hc.logger).Log("msg", "unknown event type")
		return
	}

	level.Debug(hc.logger).Log("function", "onDelete", "msg", sg.ObjectMeta.SelfLink)

	deploymentsClient := hc.config.KubernetesClientset.AppsV1beta1Client.Deployments(sg.ObjectMeta.Namespace)
	deploymentName := sg.Name

	// With this policy, dependent resources will be deleted, but we don't wait
	// for that to happen.
	deletePolicy := metav1.DeletePropagationBackground
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	err := deploymentsClient.Delete(deploymentName, deleteOptions)
	if err != nil {
		level.Error(hc.logger).Log("msg", err)
		return
	}

	level.Info(hc.logger).Log("msg", "deleted deployment", "name", deploymentName)
}

func (hc *HabitatController) watchPods(ctx context.Context) {
	ls := labels.SelectorFromSet(labels.Set(map[string]string{"habitat": "true"}))
	clw := newListWatchFromClientWithLabels(
		hc.config.KubernetesClientset.CoreV1().RESTClient(),
		"pods",
		apiv1.NamespaceAll,
		ls)

	_, c := cache.NewInformer(
		clw,
		&apiv1.Pod{},
		resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    hc.onPodAdd,
			UpdateFunc: hc.onPodUpdate,
			DeleteFunc: hc.onPodDelete,
		})

	go c.Run(ctx.Done())
}

func (hc *HabitatController) onPodAdd(obj interface{}) {
}

func (hc *HabitatController) onPodUpdate(oldObj, newObj interface{}) {
	// TODO: Do not retrieve or write IP if we are deploying a standalone topology.
	pod, ok := newObj.(*apiv1.Pod)
	if !ok {
		level.Error(hc.logger).Log("msg", "Unknown event received in Pod Update handler")
		return
	}

	if pod == nil {
		level.Warn(hc.logger).Log("msg", "Empty event received in Pod Update handler")
		return
	}

	if pod.Status.Phase != apiv1.PodRunning {
		return
	}
	err := hc.writeIP(pod)
	if err != nil {
		level.Error(hc.logger).Log("msg", err)
		return
	}
}

func (hc *HabitatController) onPodDelete(obj interface{}) {
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		level.Error(hc.logger).Log("msg", "Unknown event received in Pod Delete handler")
		return
	}

	if pod == nil {
		level.Warn(hc.logger).Log("msg", "Empty event received in Pod Delete handler")
		return
	}

	sgName, exists := pod.ObjectMeta.Labels["service-group"]
	if !exists {
		level.Error(hc.logger).Log("msg", "Could not retrieve service group name because label does not exist.")
		return
	}

	cmName := configMapName(sgName)

	cm, err := hc.config.KubernetesClientset.CoreV1().ConfigMaps(apiv1.NamespaceDefault).Get(cmName, metav1.GetOptions{})
	if err != nil {
		level.Error(hc.logger).Log("msg", err)
		return
	}

	currIP := cm.Data[peerFile]

	deletedPodIP := pod.Status.PodIP
	if currIP != deletedPodIP {
		return
	}

	// Get only running pods.
	fs := fields.SelectorFromSet(fields.Set{
		"status.phase": "Running",
	})

	podList, err := hc.config.KubernetesClientset.CoreV1().Pods(apiv1.NamespaceDefault).List(metav1.ListOptions{FieldSelector: fs.String()})
	if err != nil {
		level.Error(hc.logger).Log("msg", err)
		return
	}

	for _, newPod := range podList.Items {
		if newPod.Status.Phase == apiv1.PodRunning {
			// Replace our IP in the CM file with a new IP of a running pod.
			err := hc.writeIP(&newPod)
			if err != nil {
				level.Error(hc.logger).Log("msg", err)
			}
			return
		}
	}
}

func (hc *HabitatController) writeIP(pod *apiv1.Pod) error {
	sgName := pod.ObjectMeta.Labels["service-group"]
	cmName := configMapName(sgName)
	ip := pod.Status.PodIP

	cm, err := hc.config.KubernetesClientset.CoreV1().ConfigMaps(apiv1.NamespaceDefault).Get(cmName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	oldIP := cm.Data[peerFile]
	if oldIP != "" {
		// Do not overwrite IP with itself.
		if ip == oldIP {
			return nil
		}
		podList, err := hc.config.KubernetesClientset.CoreV1().Pods(apiv1.NamespaceDefault).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, oldPod := range podList.Items {
			// Do not write a new IP if pod with the IP in the CM is still running.
			if oldPod.Status.PodIP == oldIP && oldPod.Status.Phase == apiv1.PodRunning {
				return nil
			}
		}
	}

	// We need to retrieve our deployment to get the UID for the OwnerReference.
	d, err := hc.config.KubernetesClientset.AppsV1beta1Client.Deployments(apiv1.NamespaceDefault).Get(sgName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	updatedCM := newConfigMap(sgName, d.UID, ip)
	_, err = hc.config.KubernetesClientset.CoreV1().ConfigMaps(apiv1.NamespaceDefault).Update(updatedCM)
	if err != nil {
		return err
	}
	return nil
}

func newConfigMap(sgName string, parentUID types.UID, ip string) *apiv1.ConfigMap {
	return &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapName(sgName),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "extensions/v1beta1",
					Kind:       "Deployment",
					Name:       sgName,
					UID:        parentUID,
				},
			},
		},
		Data: map[string]string{
			peerFile: ip,
		},
	}
}

func configMapName(sgName string) string {
	return fmt.Sprintf("%s-peer-file", sgName)
}
