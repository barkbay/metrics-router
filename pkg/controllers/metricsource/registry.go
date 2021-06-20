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

package metricsource

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/barkbay/custom-metrics-router/pkg/api/v1alpha1"
	"github.com/barkbay/custom-metrics-router/pkg/metricsclient"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

type MetricsServiceBackendState struct {
	priority            int
	customMetricInfos   map[provider.CustomMetricInfo]struct{}
	externalMetricInfos map[provider.ExternalMetricInfo]struct{}
	client              *metricsclient.Client
}

type Registry struct {
	mapper     meta.RESTMapper
	baseConfig *rest.Config
	lock       sync.RWMutex

	// metricsServiceBackendState holds the current state of the metrics served by a metric backend.
	// Key is the name of the metric source
	metricsServiceBackendState map[types.NamespacedName]MetricsServiceBackendState

	customMetrics   map[provider.CustomMetricInfo]*MetricsServices
	externalMetrics map[provider.ExternalMetricInfo]*MetricsServices
}

// TODO anaik: Refactor so that the old client can be reused when nothing changes.
func (r *Registry) AddOrUpdateService(source v1alpha1.MetricsSource) (int, error) {
	metricCount := 0
	client, err := metricsclient.NewClient(
		source.Spec.InsecureSkipTLSVerify,
		source.Spec.MetricsServiceBackend,
		r.baseConfig,
		r.mapper,
	)
	if err != nil {
		return metricCount, err
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	// Read custom metrics available from this metric source
	customMetricInfos := make(map[provider.CustomMetricInfo]struct{})
	if source.Spec.MetricTypes.HasCustomMetrics() {
		customMetricInfos, err = client.ListCustomMetricInfos()
		if err != nil {
			return metricCount, fmt.Errorf("failed to list custom metric api resources: %v", err)
		}
		metricCount += len(customMetricInfos)
	}
	if backendState, ok := r.metricsServiceBackendState[source.NamespacedNamed()]; ok {
		oldMetricInfos := getOldCustomMetricInfos(backendState.customMetricInfos, customMetricInfos)
		for _, outdated := range oldMetricInfos {
			r.customMetrics[outdated].RemoveSource(source.NamespacedNamed())
		}
	}
	for mInfo := range customMetricInfos {
		var ok bool
		if _, ok = r.customMetrics[mInfo]; !ok {
			r.customMetrics[mInfo] = NewMetricServiceList()
		}
		serviceList := r.customMetrics[mInfo]
		serviceList.AddOrUpdateSource(source)
	}

	externalMetricInfos := make(map[provider.ExternalMetricInfo]struct{})
	if source.Spec.MetricTypes.HasExternalMetrics() {
		externalMetricInfos, err = client.ListExternalMetrics()
		if err != nil {
			return metricCount, fmt.Errorf("failed to list external metric api resources: %v", err)
		}
		metricCount += len(externalMetricInfos)
	}
	if backendState, ok := r.metricsServiceBackendState[source.NamespacedNamed()]; ok {
		oldMetricInfos := getOldExternalMetricInfos(backendState.externalMetricInfos, externalMetricInfos)
		for _, outdated := range oldMetricInfos {
			r.externalMetrics[outdated].RemoveSource(source.NamespacedNamed())
		}
	}
	for mInfo := range externalMetricInfos {
		var ok bool
		if _, ok = r.externalMetrics[mInfo]; !ok {
			r.externalMetrics[mInfo] = NewMetricServiceList()
		}
		serviceList := r.externalMetrics[mInfo]
		serviceList.AddOrUpdateSource(source)
	}

	// Update the state for this backend
	r.metricsServiceBackendState[source.NamespacedNamed()] = MetricsServiceBackendState{
		priority:            source.Spec.Priority,
		client:              client,
		customMetricInfos:   customMetricInfos,
		externalMetricInfos: externalMetricInfos,
	}
	return metricCount, nil
}

func getOldCustomMetricInfos(old map[provider.CustomMetricInfo]struct{}, new map[provider.CustomMetricInfo]struct{}) []provider.CustomMetricInfo {
	var outdated []provider.CustomMetricInfo
	for info := range old {
		if _, ok := new[info]; !ok {
			outdated = append(outdated, info)
		}
	}
	return outdated
}

func getOldExternalMetricInfos(old map[provider.ExternalMetricInfo]struct{}, new map[provider.ExternalMetricInfo]struct{}) []provider.ExternalMetricInfo {
	var outdated []provider.ExternalMetricInfo
	for info := range old {
		if _, ok := new[info]; !ok {
			outdated = append(outdated, info)
		}
	}
	return outdated
}

func (r *Registry) DeleteService(name types.NamespacedName) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for k, v := range r.customMetrics {
		empty := v.RemoveSource(name)
		if empty {
			delete(r.customMetrics, k)
		}
	}
	delete(r.metricsServiceBackendState, name)
}

func (r *Registry) GetMetricsBackend(info provider.CustomMetricInfo) (*metricsclient.Client, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	var services *MetricsServices
	var metricsService MetricsServiceBackendState
	var ok bool
	if services, ok = r.customMetrics[info]; !ok {
		return nil, &errors.StatusError{
			ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Code:    http.StatusNotFound,
				Reason:  metav1.StatusReasonNotFound,
				Message: fmt.Sprintf("custom metric %s is not provided by any metrics backend", info.Metric),
			}}
	}
	service, err := services.GetBestMetricService()
	if err != nil {
		return nil, fmt.Errorf("not backend for metric: %v", info.Metric)
	}
	if metricsService, ok = r.metricsServiceBackendState[service.NamespacedNamed()]; !ok {
		return nil, fmt.Errorf("properties for metric service %s/%s is missing", service.Namespace, service.Name)
	}
	return metricsService.client, nil
}
func (r *Registry) GetExternalMetricsBackend(info provider.ExternalMetricInfo) (*metricsclient.Client, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	var services *MetricsServices
	var metricsService MetricsServiceBackendState
	var ok bool
	if services, ok = r.externalMetrics[info]; !ok {
		return nil, &errors.StatusError{
			ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Code:    http.StatusNotFound,
				Reason:  metav1.StatusReasonNotFound,
				Message: fmt.Sprintf("external metric %s is not provided by any metrics backend", info.Metric),
			}}
	}
	service, err := services.GetBestMetricService()
	if err != nil {
		return nil, fmt.Errorf("not backend for metric: %v", info.Metric)
	}
	if metricsService, ok = r.metricsServiceBackendState[service.NamespacedNamed()]; !ok {
		return nil, fmt.Errorf("properties for metric service %s/%s is missing", service.Namespace, service.Name)
	}
	return metricsService.client, nil
}

func (r *Registry) ListAllCustomMetrics() []provider.CustomMetricInfo {
	r.lock.RLock()
	defer r.lock.RUnlock()
	infos := make([]provider.CustomMetricInfo, len(r.customMetrics))
	count := 0
	for k := range r.customMetrics {
		infos[count] = k
		count++
	}
	return infos
}

func (r *Registry) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	r.lock.RLock()
	defer r.lock.RUnlock()
	infos := make([]provider.ExternalMetricInfo, len(r.externalMetrics))
	count := 0
	for k := range r.externalMetrics {
		infos[count] = k
		count++
	}
	return infos
}
