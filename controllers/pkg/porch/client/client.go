// Copyright 2023 The Nephio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	pvapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	coreapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateClient(config *rest.Config) (client.Client, error) {
	scheme, err := createScheme()
	if err != nil {
		return nil, err
	}

	c, err := client.New(config, client.Options{
		Scheme: scheme,
		Mapper: createRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

// controller-runtime does not support subresources so we use REST client directly.
// TODO: Separate Porch clientset into its own module (similar to k8s clientsets) to use it
// without causing circular reference.
func CreateRESTClient(config *rest.Config) (rest.Interface, error) {
	scheme, err := createScheme()
	if err != nil {
		return nil, err
	}

	codecs := serializer.NewCodecFactory(scheme)

	gv := porchapi.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = codecs.WithoutConversion()

	return rest.RESTClientFor(config)
}

func AddToScheme(scheme *runtime.Scheme) error {
	for _, api := range (runtime.SchemeBuilder{
		configapi.AddToScheme,
		porchapi.AddToScheme,
		pvapi.AddToScheme,
	}) {
		if err := api(scheme); err != nil {
			return err
		}
	}

	return nil
}

func createScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	if err := AddToScheme(scheme); err != nil {
		return nil, err
	}
	for _, api := range (runtime.SchemeBuilder{
		coreapi.AddToScheme,
		metav1.AddMetaToScheme,
	}) {
		if err := api(scheme); err != nil {
			return nil, err
		}
	}
	return scheme, nil
}

func createRESTMapper() meta.RESTMapper {
	rm := meta.NewDefaultRESTMapper([]schema.GroupVersion{
		configapi.GroupVersion,
		porchapi.SchemeGroupVersion,
		coreapi.SchemeGroupVersion,
		metav1.SchemeGroupVersion,
	})

	for _, r := range []struct {
		kind             schema.GroupVersionKind
		plural, singular string
	}{
		{kind: configapi.GroupVersion.WithKind("Repository"), plural: "repositories", singular: "repository"},
		{kind: porchapi.SchemeGroupVersion.WithKind("PackageRevision"), plural: "packagerevisions", singular: "packagerevision"},
		{kind: porchapi.SchemeGroupVersion.WithKind("PackageRevisionResources"), plural: "packagerevisionresources", singular: "packagerevisionresources"},
		{kind: porchapi.SchemeGroupVersion.WithKind("Function"), plural: "functions", singular: "function"},
		{kind: coreapi.SchemeGroupVersion.WithKind("Secret"), plural: "secrets", singular: "secret"},
		{kind: metav1.SchemeGroupVersion.WithKind("Table"), plural: "tables", singular: "table"},
	} {
		rm.AddSpecific(
			r.kind,
			r.kind.GroupVersion().WithResource(r.plural),
			r.kind.GroupVersion().WithResource(r.singular),
			meta.RESTScopeNamespace,
		)
	}

	return rm
}
