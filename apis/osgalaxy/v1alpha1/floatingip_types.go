/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// FloatingipParameters are the configurable fields of a Floatingip.
type FloatingipParameters struct {
	Name              string `json:"name,omitempty"`
	ProjectId         string `json:"projectId,omitempty"`
	CellId            string `json:"cellId,omitempty"`
	Isp               string `json:"isp,omitempty"`
	FloatingNetworkId string `json:"floatingNetworkId,omitempty"`
	QosPolicyId       string `json:"qosPolicyId,omitempty"`
	RouteId           string `json:"routeId,omitempty"`
	Description       string `json:"description,omitempty"`
	ReservationId     string `json:"reservationId,omitempty"`
	Bandwidth         int64  `json:"bandwidth,omitempty"`
	AvailabilityZone  string `json:"availabilityZone,omitempty"`
}

// FloatingipObservation are the observable fields of a Floatingip.
type FloatingipObservation struct {
	Status string `json:"status,omitempty"`
}

// A FloatingipSpec defines the desired state of a Floatingip.
type FloatingipSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       FloatingipParameters `json:"forProvider"`
}

// A FloatingipStatus represents the observed state of a Floatingip.
type FloatingipStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          FloatingipObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Floatingip is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ucan},shortName=fipu
type Floatingip struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FloatingipSpec   `json:"spec"`
	Status FloatingipStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FloatingipList contains a list of Floatingip
type FloatingipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Floatingip `json:"items"`
}

// Floatingip type metadata.
var (
	FloatingipKind             = reflect.TypeOf(Floatingip{}).Name()
	FloatingipGroupKind        = schema.GroupKind{Group: Group, Kind: FloatingipKind}.String()
	FloatingipKindAPIVersion   = FloatingipKind + "." + SchemeGroupVersion.String()
	FloatingipGroupVersionKind = SchemeGroupVersion.WithKind(FloatingipKind)
)

func init() {
	SchemeBuilder.Register(&Floatingip{}, &FloatingipList{})
}
