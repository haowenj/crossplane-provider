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

type PersonalityParameters struct {
	Contents string `json:"contents"`
	Path     string `json:"path"`
}

type SecurityGroupParameters struct {
	Name string `json:"name"`
}

type BlockDeviceParameters struct {
	BootIndex           int64  `json:"bootIndex,omitempty"`
	DeleteOnTermination bool   `json:"deleteOnTermination,omitempty"`
	DeviceName          string `json:"deviceName"`
	SourceType          string `json:"sourceType"`
	DestinationType     string `json:"destinationType,omitempty"`
	VolumeSize          int64  `json:"volumeSize,omitempty"`
	VolumeType          string `json:"volumeType,omitempty"`
	UUID                string `json:"uuid,omitempty"`
}

// VirtualMachineParameters are the configurable fields of a VirtualMachine.
type VirtualMachineParameters struct {
	Name               string                    `json:"name,omitempty"`
	ProjectId          string                    `json:"projectId,omitempty"`
	CellId             string                    `json:"cellId,omitempty"`
	AccessIPV4         string                    `json:"accessIpV4,omitempty"`
	AccessIPV6         string                    `json:"accessIpV6,omitempty"`
	ImageRef           string                    `json:"imageRef,omitempty"`
	FlavorRef          string                    `json:"flavorRef,omitempty"`
	AvailabilityZone   string                    `json:"availabilityZone,omitempty"`
	UserData           string                    `json:"userData,omitempty"`
	BlockDeviceMapping []BlockDeviceParameters   `json:"blockDeviceMapping,omitempty"`
	Metadata           map[string]string         `json:"metadata,omitempty"`
	Personality        []PersonalityParameters   `json:"personality,omitempty"`
	SecurityGroups     []SecurityGroupParameters `json:"securityGroups,omitempty"`
}

// VirtualMachineObservation are the observable fields of a VirtualMachine.
type VirtualMachineObservation struct {
	ObservableField string `json:"observableField,omitempty"`
}

// A VirtualMachineSpec defines the desired state of a VirtualMachine.
type VirtualMachineSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       VirtualMachineParameters `json:"forProvider"`
}

// A VirtualMachineStatus represents the observed state of a VirtualMachine.
type VirtualMachineStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          VirtualMachineObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A VirtualMachine is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,ucan}
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualMachineList contains a list of VirtualMachine
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

// VirtualMachine type metadata.
var (
	VirtualMachineKind             = reflect.TypeOf(VirtualMachine{}).Name()
	VirtualMachineGroupKind        = schema.GroupKind{Group: Group, Kind: VirtualMachineKind}.String()
	VirtualMachineKindAPIVersion   = VirtualMachineKind + "." + SchemeGroupVersion.String()
	VirtualMachineGroupVersionKind = SchemeGroupVersion.WithKind(VirtualMachineKind)
)

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}
