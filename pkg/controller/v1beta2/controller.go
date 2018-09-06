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

package v1beta2

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sync"
	"time"

	habv1beta1 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta1"
	habscheme "github.com/habitat-sh/habitat-operator/pkg/client/clientset/versioned/scheme"
	habinformers "github.com/habitat-sh/habitat-operator/pkg/client/informers/externalversions"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	resyncPeriod = 1 * time.Minute

	userTOMLFile = "user.toml"
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

	userConfigFilename = "user-config"

	controllerAgentName = "habitat-controller"

	// Events.
	validationFailed = "ValidationFailed"
	cmCreated        = "ConfigMapCreated"
	cmUpdated        = "ConfigMapUpdated"
	cmFailed         = "ConfigMapCreationFailed"
	stsCreated       = "StatefulSetCreated"
	stsFailed        = "StatefulSetCreationFailed"

	// Event messages.
	messageValidationFailed = "Failed validating Habitat"
	messageCMCreated        = "Created peer IP ConfigMap"
	messageCMUpdated        = "Updated peer IP ConfigMap"
	messageCMFailed         = "Failed creating ConfigMap"
	messagePeerIPAdded      = "Added peer IP to ConfigMap"
	messagePeerIPUpdated    = "Updated peer IP in ConfigMap"
	messagePeerIPRemoved    = "Removed peer IP from ConfigMap"
	messageStsCreated       = "Created StatefulSet"
	messageStsFailed        = "Failed creating StatefulSet"
)

var ringRegexp *regexp.Regexp = regexp.MustCompile(ringKeyRegexp)

type HabitatController struct {
	config Config
	logger log.Logger

	// queue contains the jobs that will be handled by syncHabitat.
	// A workqueue.RateLimitingInterface is a queue where failing jobs are re-enqueued with an exponential
	// delay, so that jobs in a crashing loop don't fill the queue.
	queue workqueue.RateLimitingInterface

	habInformer cache.SharedIndexInformer
	stsInformer cache.SharedIndexInformer
	cmInformer  cache.SharedIndexInformer

	// cache.InformerSynced returns true if the store has been synced at least once.
	habInformerSynced cache.InformerSynced
	stsInformerSynced cache.InformerSynced
	cmInformerSynced  cache.InformerSynced

	recorder record.EventRecorder
}

type Config struct {
	HabitatClient          rest.Interface
	KubernetesClientset    *kubernetes.Clientset
	ClusterConfig          *rest.Config
	KubeInformerFactory    kubeinformers.SharedInformerFactory
	HabitatInformerFactory habinformers.SharedInformerFactory
	Namespace              string
}

func New(config Config, logger log.Logger) (*HabitatController, error) {
	if config.HabitatClient == nil {
		return nil, errors.New("invalid controller config: no HabitatClient")
	}
	if config.KubernetesClientset == nil {
		return nil, errors.New("invalid controller config: no KubernetesClientset")
	}
	if config.KubeInformerFactory == nil {
		return nil, errors.New("invalid controller config: no KubeInformerFactory")
	}
	if config.HabitatInformerFactory == nil {
		return nil, errors.New("invalid controller config: no HabitatInformerFactory")
	}
	if logger == nil {
		return nil, errors.New("invalid controller config: no logger")
	}

	// Set up event broadcasting.
	habscheme.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: config.KubernetesClientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: controllerAgentName})

	hc := &HabitatController{
		config:   config,
		logger:   logger,
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Habitats"),
		recorder: recorder,
	}

	return hc, nil
}

// Run starts a Habitat resource controller.
func (hc *HabitatController) Run(ctx context.Context, workers int) error {
	level.Info(hc.logger).Log("msg", "Watching Habitat objects")

	var wg sync.WaitGroup
	wg.Add(4 + workers)

	hc.cacheHabitats()
	hc.cacheStatefulSets()
	hc.cacheConfigMaps()
	hc.watchPods(ctx, &wg)

	go func() {
		hc.habInformer.Run(ctx.Done())
		wg.Done()
	}()

	go func() {
		hc.stsInformer.Run(ctx.Done())
		wg.Done()
	}()

	go func() {
		hc.cmInformer.Run(ctx.Done())
		wg.Done()
	}()

	// Wait for caches to be synced before starting workers.
	if !cache.WaitForCacheSync(ctx.Done(), hc.habInformerSynced, hc.stsInformerSynced, hc.cmInformerSynced) {
		return nil
	}
	level.Debug(hc.logger).Log("msg", "Caches synced")

	// Start the synchronous queue consumers. If a worker exits because of a
	// failed job, it will be restarted after a delay of 1 second.
	for i := 0; i < workers; i++ {
		level.Debug(hc.logger).Log("msg", "Starting worker", "id", i)
		go func() {
			wait.Until(hc.worker, time.Second, ctx.Done())
			wg.Done()
		}()
	}

	// This channel is closed when the context is canceled or times out.
	<-ctx.Done()

	// Make sure the work queue is shutdown which will trigger workers to end.
	hc.queue.ShutDown()

	// Block until the WaitGroup counter is zero
	wg.Wait()

	// Err() contains the error, if any.
	return ctx.Err()
}

func (hc *HabitatController) cacheHabitats() {
	hc.habInformer = hc.config.HabitatInformerFactory.Habitat().V1beta1().Habitats().Informer()

	hc.habInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    hc.handleHabAdd,
		UpdateFunc: hc.handleHabUpdate,
		DeleteFunc: hc.handleHabDelete,
	})

	hc.habInformerSynced = hc.habInformer.HasSynced
}

func (hc *HabitatController) cacheConfigMaps() {
	hc.cmInformer = hc.config.KubeInformerFactory.Core().V1().ConfigMaps().Informer()

	hc.cmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    hc.handleCMAdd,
		UpdateFunc: hc.handleCMUpdate,
		DeleteFunc: hc.handleCMDelete,
	})

	hc.cmInformerSynced = hc.cmInformer.HasSynced
}

func (hc *HabitatController) watchPods(ctx context.Context, wg *sync.WaitGroup) {
	source := cache.NewFilteredListWatchFromClient(
		hc.config.KubernetesClientset.CoreV1().RESTClient(),
		"Pods",
		hc.config.Namespace,
		listOptions())

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

	go func() {
		c.Run(ctx.Done())
		wg.Done()
	}()
}

func (hc *HabitatController) handleHabAdd(obj interface{}) {
	h, ok := obj.(*habv1beta1.Habitat)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", obj)
		return
	}

	hc.enqueue(h)
}

func (hc *HabitatController) handleHabUpdate(oldObj, newObj interface{}) {
	oldHab, ok := oldObj.(*habv1beta1.Habitat)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", oldObj)
		return
	}

	newHab, ok := newObj.(*habv1beta1.Habitat)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", newObj)
		return
	}

	if hc.habitatNeedsUpdate(oldHab, newHab) {
		hc.enqueue(newHab)
	}
}

func (hc *HabitatController) handleHabDelete(obj interface{}) {
	h, ok := obj.(*habv1beta1.Habitat)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", obj)
		return
	}

	hc.enqueue(h)
}

func (hc *HabitatController) handleCM(obj interface{}) {
	cm, ok := obj.(*apiv1.ConfigMap)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert ConfigMap", "obj", obj)
		return
	}

	cache.ListAll(hc.habInformer.GetStore(), labels.Everything(), func(obj interface{}) {
		h, ok := obj.(*habv1beta1.Habitat)
		if !ok {
			level.Error(hc.logger).Log("msg", "Failed to type assert Habitat", "obj", obj)
			return
		}

		if h.Namespace == cm.GetNamespace() {
			hc.enqueue(h)
		}
	})
}

func (hc *HabitatController) handleCMAdd(obj interface{}) {
	hc.handleCM(obj)
}

func (hc *HabitatController) handleCMUpdate(oldObj, newObj interface{}) {
	hc.handleCM(newObj)
}

func (hc *HabitatController) handleCMDelete(obj interface{}) {
	hc.handleCM(obj)
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
		if hErr, ok := err.(keyNotFoundError); !ok {
			level.Error(hc.logger).Log("msg", hErr, "key", hErr.key)
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
		if hErr, ok := err.(keyNotFoundError); !ok {
			level.Error(hc.logger).Log("msg", hErr, "key", hErr.key)
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
		"status.phase": string(apiv1.PodRunning),
	})
	ls := fields.SelectorFromSet(fields.Set(map[string]string{
		habv1beta1.HabitatLabel: "true",
	}))

	running := metav1.ListOptions{
		FieldSelector: fs.String(),
		LabelSelector: ls.String(),
	}

	pods, err := hc.config.KubernetesClientset.CoreV1().Pods(namespace).List(running)
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

func (hc *HabitatController) handleConfigMap(h *habv1beta1.Habitat) error {
	runningPods, err := hc.getRunningPods(h.Namespace)
	if err != nil {
		return err
	}

	if len(runningPods) == 0 {
		// No running Pods, create an empty ConfigMap.
		newCM := newConfigMap("", h)

		cm, err := hc.config.KubernetesClientset.CoreV1().ConfigMaps(h.Namespace).Create(newCM)
		if err != nil {
			// Was the error due to the ConfigMap already existing?
			if !apierrors.IsAlreadyExists(err) {
				return err
			}

			// Find and delete the IP in the existing ConfigMap:
			// it must necessarily be invalid, since there are no running Pods.
			cm, err = hc.findConfigMapInCache(newCM)
			if err != nil {
				return err
			}

			if err := hc.writeLeaderIP(cm, ""); err != nil {
				return err
			}

			level.Debug(hc.logger).Log("msg", messagePeerIPRemoved, "name", newCM.Name)
			hc.recorder.Event(h, apiv1.EventTypeNormal, cmUpdated, messagePeerIPRemoved)

			return nil
		}

		level.Info(hc.logger).Log("msg", messageCMCreated, "name", cm.Name)
		hc.recorder.Event(h, apiv1.EventTypeNormal, cmCreated, messageCMCreated)

		return nil
	}

	// There are running Pods, add the IP of one of them to the ConfigMap.
	leaderIP := runningPods[0].Status.PodIP

	newCM := newConfigMap(leaderIP, h)

	cm, err := hc.config.KubernetesClientset.CoreV1().ConfigMaps(h.Namespace).Create(newCM)
	if err != nil {
		// Was the error due to the ConfigMap already existing?
		if !apierrors.IsAlreadyExists(err) {
			return err
		}

		// The ConfigMap already exists. Retrieve it and find out if the the leader
		// is still running.
		cm, err := hc.findConfigMapInCache(newCM)
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

		level.Info(hc.logger).Log("msg", messagePeerIPUpdated, "name", cm.Name, "ip", leaderIP)
		hc.recorder.Event(h, apiv1.EventTypeNormal, cmUpdated, messagePeerIPUpdated)
	} else {
		level.Info(hc.logger).Log("msg", messageCMCreated, "name", cm.Name, "ip", leaderIP)
		hc.recorder.Event(h, apiv1.EventTypeNormal, cmCreated, messageCMCreated)
	}

	return nil
}

func (hc *HabitatController) enqueue(hab *habv1beta1.Habitat) {
	if hab == nil {
		level.Error(hc.logger).Log("msg", "Habitat object was nil", "object", hab)
		return
	}

	if err := checkCustomVersionMatch(hab.CustomVersion); err != nil {
		level.Info(hc.logger).Log("msg", err)
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
	// Process an item, unless `queue.ShutDown()` has been called, in which case we exit.
	key, quit := hc.queue.Get()
	if quit {
		return false
	}

	k, ok := key.(string)
	if !ok {
		level.Error(hc.logger).Log("msg", "Failed to type assert key", "obj", key)

		// The item in the queue does not have the expected type, so we remove it
		// from the queue as there's no point in processing it.
		hc.queue.Forget(k)

		return false
	}

	defer hc.queue.Done(key)

	if err := hc.conform(k); err != nil {
		level.Error(hc.logger).Log("msg", "Habitat could not be synced, requeueing", "err", err, "obj", k)

		hc.queue.AddRateLimited(k)

		return true
	}

	// If there was no error, tell the queue it can stop tracking failure history for the key.
	hc.queue.Forget(k)

	return true
}

// conform is where the reconciliation takes place.
// It is invoked when any of the following resources get created, updated or deleted:
// Habitat, Pod, StatefulSet, ConfigMap.
func (hc *HabitatController) conform(key string) error {
	obj, exists, err := hc.habInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		// The Habitat was deleted.
		level.Info(hc.logger).Log("msg", "deleted Habitat", "key", key)
		return nil
	}

	// The Habitat was either created or updated.
	h, ok := obj.(*habv1beta1.Habitat)
	if !ok {
		return fmt.Errorf("unknown event type")
	}

	level.Debug(hc.logger).Log("function", "handle Habitat Creation", "msg", h.ObjectMeta.SelfLink)

	// Validate object.
	if err := validateCustomObject(*h); err != nil {
		hc.recorder.Event(h, apiv1.EventTypeWarning, validationFailed, messageValidationFailed)
		return err
	}

	level.Debug(hc.logger).Log("msg", "validated object")

	newSts, err := hc.newStatefulSet(h)
	if err != nil {
		hc.recorder.Eventf(h, apiv1.EventTypeWarning, stsFailed, fmt.Sprintf("%s: %s", messageStsFailed, err))
		return err
	}

	// Create StatefulSet, if it doesn't already exist.
	if _, err := hc.config.KubernetesClientset.AppsV1().StatefulSets(h.Namespace).Create(newSts); err != nil {
		// Was the error due to the StatefulSet already existing?
		if apierrors.IsAlreadyExists(err) {
			// If yes, update it but retrieve the current state before that.
			oldSts, err := hc.config.KubernetesClientset.AppsV1().StatefulSets(h.Namespace).Get(newSts.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// Update the StatefulSet
			updatedSts, err := hc.config.KubernetesClientset.AppsV1().StatefulSets(h.Namespace).Update(newSts)
			if err != nil {
				return err
			}

			// Workaround for upstream bug with the habitat supervisor.
			// https://github.com/habitat-sh/habitat/issues/5264
			//
			// When the bug is fixed and the workaround is removed, make
			// sure to change UpdateStrategy to RollingUpdate as OnDelete
			// will break updates to deployments.
			if killAllPods := !reflect.DeepEqual(oldSts.Spec.Template, updatedSts.Spec.Template); killAllPods {
				level.Info(hc.logger).Log("msg", "deleting pods under StatefulSet", "name", updatedSts.Name)
				if err := hc.deleteStatefulSetPods(updatedSts); err != nil {
					return err
				}
			}

			level.Debug(hc.logger).Log("msg", "StatefulSet already existed", "name", updatedSts.Name)
		} else {
			hc.recorder.Event(h, apiv1.EventTypeWarning, stsFailed, messageStsFailed)
			return err
		}

	} else {
		level.Info(hc.logger).Log("msg", "created StatefulSet", "name", newSts.Name)
		hc.recorder.Event(h, apiv1.EventTypeNormal, stsCreated, messageStsCreated)
	}

	// Handle creation/updating of peer IP ConfigMap.
	if err := hc.handleConfigMap(h); err != nil {
		hc.recorder.Eventf(h, apiv1.EventTypeWarning, cmFailed, fmt.Sprintf("%s: %s", messageCMFailed, err))
		return err
	}

	return nil
}

func (hc *HabitatController) habitatNeedsUpdate(oldHabitat, newHabitat *habv1beta1.Habitat) bool {
	if reflect.DeepEqual(oldHabitat.Spec.V1beta2, newHabitat.Spec.V1beta2) {
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

func (hc *HabitatController) getHabitatFromLabeledResource(r metav1.Object) (*habv1beta1.Habitat, error) {
	key, err := habitatKeyFromLabeledResource(r)
	if err != nil {
		return nil, err
	}

	obj, exists, err := hc.habInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, keyNotFoundError{key: key}
	}

	h, ok := obj.(*habv1beta1.Habitat)
	if !ok {
		return nil, fmt.Errorf("unknown object type in Habitat cache: %v", obj)
	}

	return h, nil
}

// habitatKeyFromLabeledResource returns a Store key for any resource tagged
// with the `HabitatNameLabel`.
func habitatKeyFromLabeledResource(r metav1.Object) (string, error) {
	labelName := habv1beta1.HabitatNameLabel
	hName, ok := r.GetLabels()[labelName]
	if !ok {
		return "", fmt.Errorf("Could not retrieve %q label", labelName)
	}
	if hName == "" {
		return "", fmt.Errorf("Empty value to the label: %q", labelName)
	}

	key := fmt.Sprintf("%s/%s", r.GetNamespace(), hName)
	return key, nil
}

// newConfigMap takes in ip and the habitat object and creates configmap
// using it. The name of the configmap is fixed.
func newConfigMap(ip string, h *habv1beta1.Habitat) *apiv1.ConfigMap {
	return &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: h.Namespace,
			Labels: map[string]string{
				habv1beta1.HabitatLabel: "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: habv1beta1.SchemeGroupVersion.String(),
					Kind:       habv1beta1.HabitatKind,
					Name:       h.Name,
					UID:        h.UID,
				},
			},
		},
		Data: map[string]string{
			peerFile: ip,
		},
	}
}

func isHabitatObject(objMeta *metav1.ObjectMeta) bool {
	return objMeta.Labels[habv1beta1.HabitatLabel] == "true"
}

func (hc *HabitatController) findConfigMapInCache(cm *apiv1.ConfigMap) (*apiv1.ConfigMap, error) {
	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(cm)
	if err != nil {
		level.Error(hc.logger).Log("msg", "ConfigMap key could not be retrieved", "name", cm)
		return nil, err
	}

	obj, exists, err := hc.cmInformer.GetStore().GetByKey(k)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, keyNotFoundError{key: k}
	}

	return obj.(*apiv1.ConfigMap), nil
}

func (hc *HabitatController) deleteStatefulSetPods(sts *appsv1.StatefulSet) error {
	fs := fields.SelectorFromSet(fields.Set(sts.Spec.Selector.MatchLabels))

	listOptions := metav1.ListOptions{
		LabelSelector: fs.String(),
	}

	return hc.config.KubernetesClientset.CoreV1().
		Pods(sts.Namespace).
		DeleteCollection(&metav1.DeleteOptions{}, listOptions)
}
