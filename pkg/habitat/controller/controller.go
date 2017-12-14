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
	"reflect"
	"regexp"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	appsv1beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	crv1 "github.com/kinvolk/habitat-operator/pkg/habitat/apis/cr/v1"
)

const (
	resyncPeriod = 1 * time.Minute

	userTomlFile = "user.toml"
	configMapDir = "/habitat-operator"

	peerFilename  = "peer-ip"
	peerFile      = "peer-watch-file"
	configMapName = peerFile

	// The key under which the ring key is stored in the Kubernetes Secret.
	ringSecretKey = "ring-key"
	// The extension of the key file.
	ringKeyFileExt = "sym.key"
	// Keys are saved to disk with the format `<name>-<revision>.<extension>`.
	// This regexp captures the name part.
	ringKeyRegexp = `^([\w_-]+)-\d{14}$`
)

var ringRegexp *regexp.Regexp = regexp.MustCompile(ringKeyRegexp)

type HabitatController struct {
	config Config
	logger log.Logger

	// queue contains the jobs that will be handled by syncHabitat.
	// A workqueue.RateLimitingInterface is a queue where failing jobs are re-enqueued with an exponential
	// delay, so that jobs in a crashing loop don't fill the queue.
	queue workqueue.RateLimitingInterface

	habInformer    cache.SharedIndexInformer
	deployInformer cache.SharedIndexInformer
	cMInformer     cache.SharedIndexInformer
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
		queue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "habitat"),
	}

	return hc, nil
}

// Run starts a Habitat resource controller.
func (hc *HabitatController) Run(ctx context.Context) error {
	level.Info(hc.logger).Log("msg", "Watching Habitat objects")

	hc.cacheHab()
	hc.cacheDeployment()
	hc.cacheConfigMap()
	hc.watchPods(ctx)

	go hc.habInformer.Run(ctx.Done())
	go hc.deployInformer.Run(ctx.Done())
	go hc.cMInformer.Run(ctx.Done())

	// Start the synchronous queue consumer.
	go hc.worker()

	// This channel is closed when the context is canceled or times out.
	<-ctx.Done()

	// Err() contains the error, if any.
	return ctx.Err()
}

func (hc *HabitatController) cacheHab() {
	source := cache.NewListWatchFromClient(
		hc.config.HabitatClient,
		crv1.HabitatResourcePlural,
		apiv1.NamespaceAll,
		fields.Everything())

	hc.habInformer = cache.NewSharedIndexInformer(
		source,

		// The object type.
		&crv1.Habitat{},
		resyncPeriod,
		cache.Indexers{},
	)

	hc.habInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    hc.handleHabAdd,
		UpdateFunc: hc.handleHabUpdate,
		DeleteFunc: hc.handleHabDelete,
	})
}

func (hc *HabitatController) cacheDeployment() {
	source := cache.NewListWatchFromClient(
		hc.config.KubernetesClientset.AppsV1beta1().RESTClient(),
		"deployments",
		apiv1.NamespaceAll,
		fields.Everything())

	hc.deployInformer = cache.NewSharedIndexInformer(
		source,
		&appsv1beta1.Deployment{},
		resyncPeriod,
		cache.Indexers{},
	)

	hc.deployInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    hc.handleDeployAdd,
		UpdateFunc: hc.handleDeployUpdate,
		DeleteFunc: hc.handleDeployDelete,
	})
}

func (hc *HabitatController) cacheConfigMap() {
	ls := labels.SelectorFromSet(labels.Set(map[string]string{
		crv1.HabitatLabel: "true",
	}))

	source := newListWatchFromClientWithLabels(
		hc.config.KubernetesClientset.CoreV1().RESTClient(),
		"configmaps",
		apiv1.NamespaceAll,
		ls)

	hc.cMInformer = cache.NewSharedIndexInformer(
		source,
		&apiv1.ConfigMap{},
		resyncPeriod,
		cache.Indexers{},
	)

	hc.cMInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    hc.handleCMAdd,
		UpdateFunc: hc.handleCMUpdate,
		DeleteFunc: hc.handleCMDelete,
	})
}

func (hc *HabitatController) watchPods(ctx context.Context) {
	ls := labels.SelectorFromSet(labels.Set(map[string]string{crv1.HabitatLabel: "true"}))

	source := newListWatchFromClientWithLabels(
		hc.config.KubernetesClientset.CoreV1().RESTClient(),
		"pods",
		apiv1.NamespaceAll,
		ls)

	c := cache.NewSharedIndexInformer(
		source,
		&apiv1.Pod{},
		resyncPeriod,
		cache.Indexers{},
	)

	c.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    hc.handlePodAdd,
		UpdateFunc: hc.handlePodUpdate,
		DeleteFunc: hc.handlePodDelete,
	})

	go c.Run(ctx.Done())
}

func (hc *HabitatController) handleHabAdd(obj interface{}) {
	h, ok := obj.(*crv1.Habitat)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", obj)
		return
	}

	hc.enqueue(h)
}

func (hc *HabitatController) handleHabUpdate(oldObj, newObj interface{}) {
	oldHab, ok := oldObj.(*crv1.Habitat)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", oldObj)
		return
	}

	newHab, ok := newObj.(*crv1.Habitat)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", newObj)
		return
	}

	if hc.habitatNeedsUpdate(oldHab, newHab) {
		hc.enqueue(newHab)
	}
}

func (hc *HabitatController) handleHabDelete(obj interface{}) {
	h, ok := obj.(*crv1.Habitat)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", obj)
		return
	}

	hc.enqueue(h)
}

func (hc *HabitatController) handleDeployAdd(obj interface{}) {
	d, ok := obj.(*appsv1beta1.Deployment)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert Deployment", "obj", obj)
		return
	}

	if isHabitatObject(&d.ObjectMeta) {
		h, err := hc.getHabitatFromLabeledResource(d)
		if err != nil {
			level.Error(hc.logger).Log("msg", "Could not find Habitat for Deployment", "name", d.Name)
			return
		}

		hc.enqueue(h)
	}
}

func (hc *HabitatController) handleDeployUpdate(oldObj, newObj interface{}) {
	d, ok := newObj.(*appsv1beta1.Deployment)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert deployment", "obj", newObj)
		return
	}

	if isHabitatObject(&d.ObjectMeta) {
		h, err := hc.getHabitatFromLabeledResource(d)
		if err != nil {
			level.Error(hc.logger).Log("msg", "Could not find Habitat for Deployment", "name", d.Name)
			return
		}

		hc.enqueue(h)
	}
}

func (hc *HabitatController) handleDeployDelete(obj interface{}) {
	d, ok := obj.(*appsv1beta1.Deployment)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert deployment", "obj", obj)
		return
	}

	if isHabitatObject(&d.ObjectMeta) {
		h, err := hc.getHabitatFromLabeledResource(d)
		if err != nil {
			// Could not find Habitat, it must have already been removed.
			level.Debug(hc.logger).Log("msg", "Could not find Habitat for Deployment", "name", d.Name)
			return
		}

		hc.enqueue(h)
	}
}

func (hc *HabitatController) enqueueCM(obj interface{}) {
	cm, ok := obj.(*apiv1.ConfigMap)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert ConfigMap", "obj", obj)
		return
	}

	if isHabitatObject(&cm.ObjectMeta) {
		cache.ListAll(hc.habInformer.GetStore(), labels.Everything(), func(obj interface{}) {
			h, ok := obj.(*crv1.Habitat)
			if !ok {
				level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", obj)
				return
			}
			if h.Namespace == cm.GetNamespace() {
				hc.enqueue(h)
			}
		})
	}
}

func (hc *HabitatController) handleCMAdd(obj interface{}) {
	hc.enqueueCM(obj)
}

func (hc *HabitatController) handleCMUpdate(oldObj, newObj interface{}) {
	hc.enqueueCM(newObj)
}

func (hc *HabitatController) handleCMDelete(obj interface{}) {
	hc.enqueueCM(obj)
}

func (hc *HabitatController) handlePodAdd(obj interface{}) {
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert pod", "obj", obj)
		return
	}
	if isHabitatObject(&pod.ObjectMeta) {
		h, err := hc.getHabitatFromLabeledResource(pod)
		if err != nil {
			level.Error(hc.logger).Log("msg", err)
			return
		}
		hc.enqueue(h)
	}
}

func (hc *HabitatController) handlePodUpdate(oldObj, newObj interface{}) {
	oldPod, ok1 := oldObj.(*apiv1.Pod)
	if !ok1 {
		level.Error(hc.logger).Log("msg", "Failed to type assert pod", "obj", oldObj)
		return
	}

	newPod, ok2 := newObj.(*apiv1.Pod)
	if !ok2 {
		level.Error(hc.logger).Log("msg", "Failed to type assert pod", "obj", newObj)
		return
	}

	if !hc.podNeedsUpdate(oldPod, newPod) {
		return
	}

	h, err := hc.getHabitatFromLabeledResource(newPod)
	if err != nil {
		if hErr, ok := err.(habitatNotFoundError); !ok {
			level.Error(hc.logger).Log("msg", hErr)
			return
		}

		// This only means the Pod and the Habitat watchers are not in sync.
		level.Debug(hc.logger).Log("msg", "Habitat not found", "function", "handlePodUpdate")

		return
	}

	hc.enqueue(h)
}

func (hc *HabitatController) handlePodDelete(obj interface{}) {
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert pod", "obj", obj)
		return
	}

	if !isHabitatObject(&pod.ObjectMeta) {
		return
	}

	h, err := hc.getHabitatFromLabeledResource(pod)
	if err != nil {
		if hErr, ok := err.(habitatNotFoundError); !ok {
			level.Error(hc.logger).Log("msg", hErr)
			return
		}

		// This only means the Pod and the Habitat watchers are not in sync.
		level.Debug(hc.logger).Log("msg", "Habitat not found", "function", "handlePodDelete")

		return
	}

	hc.enqueue(h)
}

func (hc *HabitatController) getRunningPods(namespace string) ([]apiv1.Pod, error) {
	fs := fields.SelectorFromSet(fields.Set{
		"status.phase": "Running",
	})
	ls := fields.SelectorFromSet(fields.Set(map[string]string{
		crv1.HabitatLabel: "true",
	}))

	running := metav1.ListOptions{
		FieldSelector: fs.String(),
		LabelSelector: ls.String(),
	}

	pods, err := hc.config.KubernetesClientset.CoreV1Client.Pods(namespace).List(running)
	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}

func (hc *HabitatController) writeLeaderIP(cm *apiv1.ConfigMap, ip string) error {
	cm.Data[peerFile] = ip

	if _, err := hc.config.KubernetesClientset.CoreV1().ConfigMaps(cm.Namespace).Update(cm); err != nil {
		return err
	}

	return nil
}

func (hc *HabitatController) handleConfigMap(h *crv1.Habitat) error {
	runningPods, err := hc.getRunningPods(h.Namespace)
	if err != nil {
		return err
	}

	if len(runningPods) == 0 {
		// No running Pods, create an empty ConfigMap.
		newCM := newConfigMap("")
		cm, err := hc.config.KubernetesClientset.CoreV1().ConfigMaps(h.Namespace).Create(newCM)
		if err != nil {
			// Was the error due to the ConfigMap already existing?
			if !apierrors.IsAlreadyExists(err) {
				return err
			}

			// Delete the IP in the existing ConfigMap, as it must necessarily be invalid,
			// since there are no running Pods.
			cm, err = hc.config.KubernetesClientset.CoreV1Client.ConfigMaps(h.Namespace).Get(newCM.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if err := hc.writeLeaderIP(cm, ""); err != nil {
				return err
			}

			level.Debug(hc.logger).Log("msg", "removed peer IP from ConfigMap", "name", newCM.Name)

			return nil
		}

		level.Info(hc.logger).Log("msg", "created peer IP ConfigMap", "name", newCM.Name)

		return nil
	}

	// There are running Pods, add the IP of one of them to the ConfigMap.
	leaderIP := runningPods[0].Status.PodIP

	newCM := newConfigMap(leaderIP)

	cm, err := hc.config.KubernetesClientset.CoreV1Client.ConfigMaps(h.Namespace).Create(newCM)
	if err != nil {
		// Was the error due to the ConfigMap already existing?
		if !apierrors.IsAlreadyExists(err) {
			return err
		}

		// The ConfigMap already exists. Is the leader still running?
		// Was the error due to the ConfigMap already existing?
		cm, err = hc.config.KubernetesClientset.CoreV1Client.ConfigMaps(h.Namespace).Get(newCM.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		curLeader := cm.Data[peerFile]

		for _, p := range runningPods {
			if p.Status.PodIP == curLeader {
				// The leader is still up, nothing to do.
				level.Debug(hc.logger).Log("msg", "Leader still running", "ip", curLeader)

				return nil
			}
		}

		// The leader is not in the list of running Pods, so the ConfigMap must be updated.
		if err := hc.writeLeaderIP(cm, leaderIP); err != nil {
			return err
		}

		level.Info(hc.logger).Log("msg", "updated peer IP in ConfigMap", "name", cm.Name, "ip", leaderIP)
	} else {
		level.Info(hc.logger).Log("msg", "created peer IP ConfigMap", "name", cm.Name, "ip", leaderIP)
	}

	return nil
}

func (hc *HabitatController) handleHabitatDeletion(key string) error {
	// Delete deployment.
	deploymentNS, deploymentName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	deploymentsClient := hc.config.KubernetesClientset.AppsV1beta1Client.Deployments(deploymentNS)

	// With this policy, dependent resources will be deleted, but we don't wait
	// for that to happen.
	deletePolicy := metav1.DeletePropagationBackground
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	if err := deploymentsClient.Delete(deploymentName, deleteOptions); err != nil {
		level.Error(hc.logger).Log("msg", err)
		return err
	}

	level.Info(hc.logger).Log("msg", "deleted deployment", "name", deploymentName)

	return nil
}

func (hc *HabitatController) newDeployment(h *crv1.Habitat) (*appsv1beta1.Deployment, error) {
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
	topology := crv1.TopologyStandalone

	if h.Spec.Service.Topology == crv1.TopologyLeader {
		topology = crv1.TopologyLeader
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

	base := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.Name,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Replicas: &count,
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						crv1.HabitatLabel:     "true",
						crv1.HabitatNameLabel: h.Name,
						crv1.TopologyLabel:    topology.String(),
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
		},
	}

	// If we have a secret name present we should mount that secret.
	if h.Spec.Service.ConfigSecretName != "" {
		// Let's make sure our secret is there before mounting it.
		secret, err := hc.config.KubernetesClientset.CoreV1().Secrets(h.Namespace).Get(h.Spec.Service.ConfigSecretName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		secretVolume := &apiv1.Volume{
			Name: "initialconfig",
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: secret.Name,
					Items: []apiv1.KeyToPath{
						{
							Key:  userTomlFile,
							Path: userTomlFile,
						},
					},
				},
			},
		}

		secretVolumeMount := &apiv1.VolumeMount{
			Name: "initialconfig",
			// Our user.toml file must be in a directory with the same name as the service.
			MountPath: fmt.Sprintf("/hab/svc/%s/%s", h.Name, userTomlFile),
			SubPath:   userTomlFile,
			ReadOnly:  false,
		}

		base.Spec.Template.Spec.Containers[0].VolumeMounts = append(base.Spec.Template.Spec.Containers[0].VolumeMounts, *secretVolumeMount)
		base.Spec.Template.Spec.Volumes = append(base.Spec.Template.Spec.Volumes, *secretVolume)
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
		base.Spec.Template.Spec.Volumes = append(base.Spec.Template.Spec.Volumes, *v)
		base.Spec.Template.Spec.Containers[0].VolumeMounts = append(base.Spec.Template.Spec.Containers[0].VolumeMounts, *vm)

		// Add --ring argument to supervisor invocation.
		base.Spec.Template.Spec.Containers[0].Args = append(base.Spec.Template.Spec.Containers[0].Args, "--ring", ringName)
	}

	return base, nil
}

func (hc *HabitatController) enqueue(hab *crv1.Habitat) {
	if hab == nil {
		level.Error(hc.logger).Log("msg", "Habitat object was nil", "object", hab)
		return
	}

	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(hab)
	if err != nil {
		level.Error(hc.logger).Log("msg", "Habitat object key could not be retrieved", "object", hab)
		return
	}

	hc.queue.Add(k)
}

func (hc *HabitatController) worker() {
	for hc.processNextItem() {
	}
}

func (hc *HabitatController) processNextItem() bool {
	key, quit := hc.queue.Get()
	if quit {
		return false
	}
	defer hc.queue.Done(key)

	k, ok := key.(string)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert key", "obj", key)
		return false
	}

	err := hc.conform(k)
	if err != nil {
		level.Error(hc.logger).Log("msg", "Habitat could not be synced, requeueing", "msg", err)

		hc.queue.AddRateLimited(k)

		return true
	}

	hc.queue.Forget(k)

	return true
}

// conform is where the reconciliation takes place.
// It is invoked when any of the following resources get created, updated or deleted:
// Habitat, Pod, Deployment, ConfigMap.
func (hc *HabitatController) conform(key string) error {
	obj, exists, err := hc.habInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		// The Habitat was deleted.
		return hc.handleHabitatDeletion(key)
	}

	// The Habitat was either created or updated.
	h, ok := obj.(*crv1.Habitat)
	if !ok {
		return fmt.Errorf("unknown event type")
	}

	level.Debug(hc.logger).Log("function", "handle Habitat Creation", "msg", h.ObjectMeta.SelfLink)

	// Validate object.
	if err := validateCustomObject(*h); err != nil {
		return err
	}

	level.Debug(hc.logger).Log("msg", "validated object")

	deployment, err := hc.newDeployment(h)
	if err != nil {
		return err
	}

	// Create Deployment, if it doesn't already exist.
	if _, err := hc.config.KubernetesClientset.AppsV1beta1Client.Deployments(h.Namespace).Create(deployment); err != nil {
		// Was the error due to the Deployment already existing?
		if apierrors.IsAlreadyExists(err) {
			// If yes, update the Deployment.
			if _, err := hc.config.KubernetesClientset.AppsV1beta1Client.Deployments(h.Namespace).Update(deployment); err != nil {
				return err
			}
		} else {
			return err
		}
		_, err = hc.config.KubernetesClientset.AppsV1beta1Client.Deployments(h.Namespace).Get(deployment.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		level.Debug(hc.logger).Log("msg", "deployment already existed", "name", deployment.Name)
	} else {
		level.Info(hc.logger).Log("msg", "created deployment", "name", deployment.Name)
	}

	// Handle creation/updating of peer IP ConfigMap.
	if err := hc.handleConfigMap(h); err != nil {
		return err
	}

	return nil
}

func (hc *HabitatController) habitatNeedsUpdate(oldHabitat, newHabitat *crv1.Habitat) bool {
	if reflect.DeepEqual(oldHabitat.Spec, newHabitat.Spec) {
		level.Debug(hc.logger).Log("msg", "Update ignored as it didn't change Habitat spec", "h", newHabitat)
		return false
	}

	return true
}

func (hc *HabitatController) podNeedsUpdate(oldPod, newPod *apiv1.Pod) bool {
	// Ignore identical objects.
	// https://github.com/kubernetes/kubernetes/blob/7e630154dfc7b2155f8946a06f92e96e268dcbcd/pkg/controller/replicaset/replica_set.go#L276-L277
	if oldPod.ResourceVersion == newPod.ResourceVersion {
		level.Debug(hc.logger).Log("msg", "Update ignored as it didn't change Pod resource version", "pod", newPod)
		return false
	}

	// Ignore changes that don't change the Pod's status.
	if oldPod.Status.Phase == newPod.Status.Phase {
		level.Debug(hc.logger).Log("msg", "Update ignored as it didn't change Pod status", "pod", newPod)
		return false
	}

	return true
}

func (hc *HabitatController) getHabitatFromLabeledResource(r metav1.Object) (*crv1.Habitat, error) {
	key, err := habitatKeyFromLabeledResource(r)
	if err != nil {
		return nil, err
	}

	obj, exists, err := hc.habInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, habitatNotFoundError{key: key}
	}

	h, ok := obj.(*crv1.Habitat)
	if !ok {
		return nil, fmt.Errorf("unknown object type in Habitat cache: %v", obj)
	}

	return h, nil
}

// habitatKeyFromLabeledResource returns a Store key for any resource tagged
// with the `HabitatNameLabel`.
func habitatKeyFromLabeledResource(r metav1.Object) (string, error) {
	hName := r.GetLabels()[crv1.HabitatNameLabel]
	if hName == "" {
		return "", fmt.Errorf("Could not retrieve %q label", crv1.HabitatNameLabel)
	}

	key := fmt.Sprintf("%s/%s", r.GetNamespace(), hName)

	return key, nil
}

func newConfigMap(ip string) *apiv1.ConfigMap {
	return &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapName,
			Labels: map[string]string{
				crv1.HabitatLabel: "true",
			},
		},
		Data: map[string]string{
			peerFile: ip,
		},
	}
}

func isHabitatObject(objMeta *metav1.ObjectMeta) bool {
	return objMeta.Labels[crv1.HabitatLabel] == "true"
}
