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

type SchedulerHintsParameters struct {
	SameHost []string `json:"sameHost,omitempty"`
}

// VolumeParameters are the configurable fields of a Volume.
type VolumeParameters struct {
	ProjectId      string                     `json:"projectId,omitempty"`
	CellId         string                     `json:"cellId,omitempty"`
	VolumeType     string                     `json:"volumeType,omitempty"`
	Size           int64                      `json:"size,omitempty"`
	Description    string                     `json:"description,omitempty"`
	Multiattach    bool                       `json:"multiattach,omitempty"`
	Name           string                     `json:"name,omitempty"`
	SchedulerHints []SchedulerHintsParameters `json:"schedulerHints,omitempty"`
}

// VolumeObservation are the observable fields of a Volume.
type VolumeObservation struct {
	ObservableField string `json:"observableField,omitempty"`
}

// A VolumeSpec defines the desired state of a Volume.
type VolumeSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       VolumeParameters `json:"forProvider"`
}

// A VolumeStatus represents the observed state of a Volume.
type VolumeStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          VolumeObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Volume is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ucan}
type Volume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeSpec   `json:"spec"`
	Status VolumeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VolumeList contains a list of Volume
type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Volume `json:"items"`
}

// Volume type metadata.
var (
	VolumeKind             = reflect.TypeOf(Volume{}).Name()
	VolumeGroupKind        = schema.GroupKind{Group: Group, Kind: VolumeKind}.String()
	VolumeKindAPIVersion   = VolumeKind + "." + SchemeGroupVersion.String()
	VolumeGroupVersionKind = SchemeGroupVersion.WithKind(VolumeKind)
)

func init() {
	SchemeBuilder.Register(&Volume{}, &VolumeList{})
}
