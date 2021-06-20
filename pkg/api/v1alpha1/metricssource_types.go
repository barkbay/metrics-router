/*
Copyright 2021.

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
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +kubebuilder:validation:Enum=CustomMetrics;ExternalMetrics

type MetricType string

var defaultBackendPort int32 = 443

// ServiceBackendPort represents an declarative configuration of the service backend to get the metrics from.
type ServiceBackendPort struct {
	Number *int32 `json:"number,omitempty"`
	// TODO: let the option to the user to specify the port by name, not only as an int
	// Name   *string `json:"name,omitempty"`
}

func (sbp ServiceBackendPort) Port() int32 {
	if sbp.Number == nil {
		return defaultBackendPort
	}
	return *sbp.Number
}

func (sbp *ServiceBackendPort) String() string {
	return strconv.Itoa(int(sbp.Port()))
}

// MetricsServiceBackend represents an declarative configuration of the MetricsServiceBackend to get the metrics from.
type MetricsServiceBackend struct {
	Namespace string             `json:"namespace,omitempty"`
	Name      string             `json:"name,omitempty"`
	Scheme    corev1.URIScheme   `json:"scheme,omitempty"`
	Port      ServiceBackendPort `json:"port,omitempty"`
}

func (m MetricsServiceBackend) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: m.Namespace,
		Name:      m.Name,
	}
}

func (m MetricsServiceBackend) scheme() corev1.URIScheme {
	if m.Scheme == "" {
		return corev1.URISchemeHTTPS
	}
	return m.Scheme
}

func (m MetricsServiceBackend) URL() string {
	return fmt.Sprintf("%s://%s.%s.svc:%d", strings.ToLower(string(m.scheme())), m.Name, m.Namespace, m.Port.Port())
}

type MetricTypes []MetricType

func (m MetricTypes) contains(metric MetricType) bool {
	for _, t := range m {
		if t == metric {
			return true
		}
	}
	return false
}

func (m MetricTypes) HasCustomMetrics() bool {
	return m.contains("CustomMetrics")
}

func (m MetricTypes) HasExternalMetrics() bool {
	return m.contains("ExternalMetrics")
}

// MetricsSourceSpec defines the desired state of MetricsSource
type MetricsSourceSpec struct {
	// Service is the K8S service to be called by the router.
	MetricsServiceBackend MetricsServiceBackend `json:"service,omitempty"`
	InsecureSkipTLSVerify bool                  `json:"insecureSkipTLSVerify,omitempty"`
	Priority              int                   `json:"priority"`
	MetricTypes           MetricTypes           `json:"metricTypes"`
}

// MetricsSourceStatus defines the observed state of MetricsSource
type MetricsSourceStatus struct {
	Synced       bool   `json:"synced"`
	MetricsCount int    `json:"metricsCount"`
	Service      string `json:"service"`
	Port         int    `json:"port"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=ms
// +kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.status.service`
// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.status.port`
// +kubebuilder:printcolumn:name="Synced",type=boolean,JSONPath=`.status.synced`
// +kubebuilder:printcolumn:name="Metrics",type=integer,JSONPath=`.status.metricsCount`

// MetricsSource is the Schema for the metricssources API
type MetricsSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetricsSourceSpec   `json:"spec,omitempty"`
	Status MetricsSourceStatus `json:"status,omitempty"`
}

// IsMarkedForDeletion returns true if the resource is going to be deleted
func (m *MetricsSource) IsMarkedForDeletion() bool {
	if m == nil {
		return false
	}
	return !m.DeletionTimestamp.IsZero()
}

func (m MetricsSource) NamespacedNamed() types.NamespacedName {
	return types.NamespacedName{
		Namespace: m.Namespace,
		Name:      m.Name,
	}
}

//+kubebuilder:object:root=true

// MetricsSourceList contains a list of MetricsSource
type MetricsSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetricsSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MetricsSource{}, &MetricsSourceList{})
}
