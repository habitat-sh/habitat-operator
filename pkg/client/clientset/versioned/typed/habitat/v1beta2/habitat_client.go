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
	v1beta2 "github.com/habitat-sh/habitat-operator/pkg/apis/habitat/v1beta2"
	"github.com/habitat-sh/habitat-operator/pkg/client/clientset/versioned/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type HabitatV1beta2Interface interface {
	RESTClient() rest.Interface
	HabitatsGetter
}

// HabitatV1beta2Client is used to interact with features provided by the habitat.sh group.
type HabitatV1beta2Client struct {
	restClient rest.Interface
}

func (c *HabitatV1beta2Client) Habitats(namespace string) HabitatInterface {
	return newHabitats(c, namespace)
}

// NewForConfig creates a new HabitatV1beta2Client for the given config.
func NewForConfig(c *rest.Config) (*HabitatV1beta2Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &HabitatV1beta2Client{client}, nil
}

// NewForConfigOrDie creates a new HabitatV1beta2Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *HabitatV1beta2Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new HabitatV1beta2Client for the given RESTClient.
func New(c rest.Interface) *HabitatV1beta2Client {
	return &HabitatV1beta2Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1beta2.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *HabitatV1beta2Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
