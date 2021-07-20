// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package managedresource

import (
	"fmt"

	resourcesv1alpha1 "github.com/gardener/gardener-resource-manager/api/resources/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type ObjectIndex struct {
	index        map[string]resourcesv1alpha1.ObjectReference
	found        sets.String
	equivalences Equivalences
}

// NewObjectIndex constructs a new *ObjectIndex containing all the given ObjectReferences. It can optionally be
// configured to use a set of rules, defining what GroupKinds to consider equivalent when looking up references
// using `Lookup()`, by passing in an `Equivalences` object. If the `Equivalences` object is nil, then references
// are only considered as equivalent if their GroupKinds are equal.
func NewObjectIndex(references []resourcesv1alpha1.ObjectReference, withEquivalences Equivalences) *ObjectIndex {
	index := &ObjectIndex{
		make(map[string]resourcesv1alpha1.ObjectReference, len(references)),
		sets.String{},
		withEquivalences,
	}

	for _, r := range references {
		index.index[objectKeyByReference(r)] = r
	}

	return index
}

// Objects returns a map containing all ObjectReferences of the index. It maps keys of the contained objects
// (in the form `Group/Kind/Namespace/Name`) to ObjectReferences.
func (i *ObjectIndex) Objects() map[string]resourcesv1alpha1.ObjectReference {
	return i.index
}

// Found checks if a given ObjectReference was found previously by a call to `Lookup()`.
func (i *ObjectIndex) Found(ref resourcesv1alpha1.ObjectReference) bool {
	return i.found.Has(objectKeyByReference(ref))
}

// Lookup checks if the index contains a given ObjectReference. It also considers cross API group equivalences
// configured by the Equivalences object handed to NewObjectIndex(). It returns the found ObjectReference and a bool
// indicating if it was found. If the reference (or equivalent one) was found it is marked as `found`, which can be
// later checked by using `Found()`.
func (i *ObjectIndex) Lookup(ref resourcesv1alpha1.ObjectReference) (resourcesv1alpha1.ObjectReference, bool) {
	key := objectKeyByReference(ref)
	if found, ok := i.index[key]; ok {
		i.found.Insert(key)
		return found, ok
	}

	gk := metav1.GroupKind{
		Group: ref.GroupVersionKind().Group,
		Kind:  ref.Kind,
	}
	if equivalenceSet := i.equivalences.GetEquivalencesFor(gk); len(equivalenceSet) > 0 {
		for equivalentGroupKind := range equivalenceSet {
			key = objectKey(equivalentGroupKind.Group, equivalentGroupKind.Kind, ref.Namespace, ref.Name)
			if found, ok := i.index[key]; ok {
				i.found.Insert(key)
				return found, ok
			}
		}
	}
	return resourcesv1alpha1.ObjectReference{}, false
}

func objectKey(group, kind, namespace, name string) string {
	if kind != "Namespace" && namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	return fmt.Sprintf("%s/%s/%s/%s", group, kind, namespace, name)
}

func objectKeyByReference(o resourcesv1alpha1.ObjectReference) string {
	return objectKey(o.GroupVersionKind().Group, o.Kind, o.Namespace, o.Name)
}