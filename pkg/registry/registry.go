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

package registry

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/barkbay/custom-metrics-router/pkg/api/v1alpha1"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

type cachedMetricSource struct {
	sourceName          string
	priority            int
	customMetricInfos   map[provider.CustomMetricInfo]struct{}
	externalMetricInfos map[provider.ExternalMetricInfo]struct{}
	client              MetricsClient
}

func NewRegistry(baseConfig *rest.Config, mapper meta.RESTMapper) *Registry {
	return &Registry{
		cachedMetricsSourcesBySource: make(map[string]cachedMetricSource),
		customMetrics:                make(map[provider.CustomMetricInfo]*cachedMetricSources),
		externalMetrics:              make(map[provider.ExternalMetricInfo]*cachedMetricSources),
		clientProvider: metricsClientProvider{
			baseConfig: baseConfig,
			mapper:     mapper,
		},
	}
}

type Registry struct {
	clientProvider MetricsClientProvider

	lock sync.RWMutex

	// cachedMetricsSourcesBySource holds the current state of the metrics served by a metric backend.
	// Key is the name of the metric source
	cachedMetricsSourcesBySource map[string]cachedMetricSource

	customMetrics   map[provider.CustomMetricInfo]*cachedMetricSources
	externalMetrics map[provider.ExternalMetricInfo]*cachedMetricSources
}

func (r *Registry) AddOrUpdateSource(source v1alpha1.MetricsSource) (int, error) {
	klog.Infof("Update metrics source %s", source.Name)
	// TODO: discuss if we should cache the client.
	client, err := r.clientProvider.NewClient(source.Spec.InsecureSkipTLSVerify, source.Spec.MetricsServiceBackend)
	if err != nil {
		return 0, err
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	// Create a new metric source
	newMetricSource := cachedMetricSource{
		sourceName:          source.Name,
		priority:            source.Spec.Priority,
		client:              client,
		customMetricInfos:   make(map[provider.CustomMetricInfo]struct{}),
		externalMetricInfos: make(map[provider.ExternalMetricInfo]struct{}),
	}

	// Read custom metrics available from this metric source
	if source.Spec.MetricTypes.HasCustomMetrics() {
		newMetricSource.customMetricInfos, err = client.ListCustomMetricInfos()
		if err != nil {
			return 0, fmt.Errorf("failed to list custom metric api resources: %v", err)
		}
	}
	if actualMetricSource, ok := r.cachedMetricsSourcesBySource[source.Name]; ok {
		// Check if some metrics that were previously served have been removed from that MetricsSource
		removedMetrics := getRemovedCustomMetrics(actualMetricSource.customMetricInfos, newMetricSource.customMetricInfos)
		for _, removedMetric := range removedMetrics {
			// This metric is more served by the metrics source
			if empty := r.customMetrics[removedMetric].removeSource(source.Name); empty {
				delete(r.customMetrics, removedMetric)
			}
		}
	}
	for mInfo := range newMetricSource.customMetricInfos {
		var ok bool
		if _, ok = r.customMetrics[mInfo]; !ok {
			r.customMetrics[mInfo] = newMetricsSources()
		}
		serviceList := r.customMetrics[mInfo]
		serviceList.addOrUpdateSource(newMetricSource)
	}

	if source.Spec.MetricTypes.HasExternalMetrics() {
		newMetricSource.externalMetricInfos, err = client.ListExternalMetrics()
		if err != nil {
			return 0, fmt.Errorf("failed to list external metric api resources: %v", err)
		}
	}
	if actualMetricSource, ok := r.cachedMetricsSourcesBySource[source.Name]; ok {
		removedMetrics := getRemovedExternalMetrics(actualMetricSource.externalMetricInfos, newMetricSource.externalMetricInfos)
		for _, removedMetric := range removedMetrics {
			if empty := r.externalMetrics[removedMetric].removeSource(source.Name); empty {
				delete(r.externalMetrics, removedMetric)
			}
		}
	}
	for mInfo := range newMetricSource.externalMetricInfos {
		var ok bool
		if _, ok = r.externalMetrics[mInfo]; !ok {
			r.externalMetrics[mInfo] = newMetricsSources()
		}
		serviceList := r.externalMetrics[mInfo]
		serviceList.addOrUpdateSource(newMetricSource)
	}

	// Update indexed cached metric sources
	r.cachedMetricsSourcesBySource[source.Name] = newMetricSource
	return len(newMetricSource.customMetricInfos) + len(newMetricSource.externalMetricInfos), nil
}

func getRemovedCustomMetrics(old map[provider.CustomMetricInfo]struct{}, new map[provider.CustomMetricInfo]struct{}) []provider.CustomMetricInfo {
	var outdated []provider.CustomMetricInfo
	for info := range old {
		if _, ok := new[info]; !ok {
			outdated = append(outdated, info)
		}
	}
	return outdated
}

func getRemovedExternalMetrics(old map[provider.ExternalMetricInfo]struct{}, new map[provider.ExternalMetricInfo]struct{}) []provider.ExternalMetricInfo {
	var outdated []provider.ExternalMetricInfo
	for info := range old {
		if _, ok := new[info]; !ok {
			outdated = append(outdated, info)
		}
	}
	return outdated
}

func (r *Registry) DeleteSource(sourceName string) {
	klog.Infof("Delete metrics source %s", sourceName)
	r.lock.Lock()
	defer r.lock.Unlock()
	// Delete related custom metrics
	for k, v := range r.customMetrics {
		if empty := v.removeSource(sourceName); empty {
			delete(r.customMetrics, k)
		}
	}
	// Delete related external metrics
	for k, v := range r.externalMetrics {
		if empty := v.removeSource(sourceName); empty {
			delete(r.externalMetrics, k)
		}
	}
	delete(r.cachedMetricsSourcesBySource, sourceName)
}

func (r *Registry) GetMetricsBackend(info provider.CustomMetricInfo) (MetricsClient, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	var services *cachedMetricSources
	var metricsService cachedMetricSource
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
	service, err := services.getBestMetricService()
	if err != nil {
		return nil, fmt.Errorf("not backend for metric: %v", info.Metric)
	}
	if metricsService, ok = r.cachedMetricsSourcesBySource[service.sourceName]; !ok {
		return nil, fmt.Errorf("properties for metric source %s is missing", service.sourceName)
	}
	klog.Infof("custom metric %v served by %s", info, metricsService.client.GetBackend().URL())
	return metricsService.client, nil
}
func (r *Registry) GetExternalMetricsBackend(info provider.ExternalMetricInfo) (MetricsClient, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	var services *cachedMetricSources
	var metricsService cachedMetricSource
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
	service, err := services.getBestMetricService()
	if err != nil {
		return nil, fmt.Errorf("not backend for metric: %v", info.Metric)
	}
	if metricsService, ok = r.cachedMetricsSourcesBySource[service.sourceName]; !ok {
		return nil, fmt.Errorf("properties for metric service %s is missing", service.sourceName)
	}
	klog.Infof("external metric %v served by %s", info, metricsService.client.GetBackend().URL())
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
