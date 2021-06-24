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
	"sync"
	"testing"

	"github.com/barkbay/custom-metrics-router/pkg/api/v1alpha1"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type fakeRegistry struct {
	registry           *Registry
	fakeClientProvider *fakeMetricsClientsProvider
}

// addCustomMetrics adds some pre-existing custom metrics in the registry
func (f *fakeRegistry) addCustomMetrics(sourceName string, priority int, metricsName ...string) *fakeRegistry {
	metricSource, ok := f.registry.cachedMetricsSourcesBySource[sourceName]
	if !ok {
		metricSource = cachedMetricSource{
			client: &fakeMetricsClient{
				name: sourceName,
				backend: v1alpha1.MetricsServiceBackend{
					Namespace: "fakens",
					Name:      sourceName,
				}},
			sourceName:          sourceName,
			priority:            priority,
			customMetricInfos:   make(map[provider.CustomMetricInfo]struct{}),
			externalMetricInfos: make(map[provider.ExternalMetricInfo]struct{}),
		}
		f.registry.cachedMetricsSourcesBySource[sourceName] = metricSource
	}

	for _, metricName := range metricsName {
		metricInfo := provider.CustomMetricInfo{Metric: metricName}
		metricSource.customMetricInfos[metricInfo] = struct{}{}

		customMetrics := f.registry.customMetrics[metricInfo]
		if customMetrics == nil {
			customMetrics = ptr(make([]cachedMetricSource, 0))
			f.registry.customMetrics[metricInfo] = customMetrics
		}
		customMetrics = ptr(append(*customMetrics, metricSource))
		f.registry.customMetrics[metricInfo] = customMetrics
	}
	return f
}

// addCustomMetrics adds some pre-existing custom metrics in the registry
func (f *fakeRegistry) addExternalMetrics(sourceName string, priority int, metricsName ...string) *fakeRegistry {
	metricSource, ok := f.registry.cachedMetricsSourcesBySource[sourceName]
	if !ok {
		metricSource = cachedMetricSource{
			client: &fakeMetricsClient{
				name: sourceName,
				backend: v1alpha1.MetricsServiceBackend{
					Namespace: "fakens",
					Name:      sourceName,
				},
			},
			sourceName:          sourceName,
			priority:            priority,
			customMetricInfos:   make(map[provider.CustomMetricInfo]struct{}),
			externalMetricInfos: make(map[provider.ExternalMetricInfo]struct{}),
		}
		f.registry.cachedMetricsSourcesBySource[sourceName] = metricSource
	}

	for _, metricName := range metricsName {
		metricInfo := provider.ExternalMetricInfo{Metric: metricName}
		metricSource.externalMetricInfos[metricInfo] = struct{}{}

		externalMetrics := f.registry.externalMetrics[metricInfo]
		if externalMetrics == nil {
			externalMetrics = ptr(make([]cachedMetricSource, 0))
			f.registry.externalMetrics[metricInfo] = externalMetrics
		}
		externalMetrics = ptr(append(*externalMetrics, metricSource))
		f.registry.externalMetrics[metricInfo] = externalMetrics
	}
	return f
}

func (f *fakeRegistry) servedCustomMetrics(source string, metricNames ...string) *fakeRegistry {
	f.fakeClientProvider.exposeCustomMetric(source, metricNames...)
	return f
}

func (f *fakeRegistry) servedExternalMetrics(source string, metricNames ...string) *fakeRegistry {
	f.fakeClientProvider.exposeExternalMetric(source, metricNames...)
	return f
}

func ptr(c cachedMetricSources) *cachedMetricSources {
	return &c
}

type fakeMetricsClient struct {
	name            string
	backend         v1alpha1.MetricsServiceBackend
	customMetrics   []string
	externalMetrics []string
}

var _ MetricsClient = &fakeMetricsClient{}

type fakeMetricsClientsProvider struct {
	clients map[string]*fakeMetricsClient
}

var _ MetricsClientProvider = &fakeMetricsClientsProvider{}

func (fmcp *fakeMetricsClientsProvider) NewClient(_ bool, msb v1alpha1.MetricsServiceBackend) (MetricsClient, error) {
	return fmcp.clients[msb.Name], nil
}

func (fmcp *fakeMetricsClientsProvider) exposeCustomMetric(sourceName string, metricsNames ...string) {
	fakeClient := fmcp.clients[sourceName]
	if fakeClient == nil {
		fakeClient = &fakeMetricsClient{
			name: sourceName,
			backend: v1alpha1.MetricsServiceBackend{
				Namespace: "fakens",
				Name:      sourceName,
			},
		}
		fmcp.clients[sourceName] = fakeClient
	}
	fakeClient.customMetrics = append(fakeClient.customMetrics, metricsNames...)
}

func (fmcp *fakeMetricsClientsProvider) exposeExternalMetric(sourceName string, metricsNames ...string) {
	fakeClient := fmcp.clients[sourceName]
	if fakeClient == nil {
		fakeClient = &fakeMetricsClient{
			name: sourceName,
			backend: v1alpha1.MetricsServiceBackend{
				Namespace: "fakens",
				Name:      sourceName,
			},
		}
		fmcp.clients[sourceName] = fakeClient
	}
	fakeClient.externalMetrics = append(fakeClient.externalMetrics, metricsNames...)
}

func (c *fakeMetricsClient) GetBackend() v1alpha1.MetricsServiceBackend {
	return c.backend
}

func (fcp *fakeMetricsClient) GetMetricByName(types.NamespacedName, provider.CustomMetricInfo, labels.Selector) (*custom_metrics.MetricValue, error) {
	panic("not implemented")
}

func (fcp *fakeMetricsClient) GetMetricBySelector(string, labels.Selector, provider.CustomMetricInfo, labels.Selector) (*custom_metrics.MetricValueList, error) {
	panic("not implemented")
}

func (fcp *fakeMetricsClient) GetExternalMetric(string, string, labels.Selector) (*external_metrics.ExternalMetricValueList, error) {
	panic("not implemented")
}

func (fcp *fakeMetricsClient) ListCustomMetricInfos() (map[provider.CustomMetricInfo]struct{}, error) {
	customMetrics := make(map[provider.CustomMetricInfo]struct{})
	for _, cm := range fcp.customMetrics {
		customMetrics[provider.CustomMetricInfo{
			GroupResource: schema.GroupResource{},
			Namespaced:    false,
			Metric:        cm,
		}] = struct{}{}
	}
	return customMetrics, nil
}

func (fcp *fakeMetricsClient) ListExternalMetrics() (map[provider.ExternalMetricInfo]struct{}, error) {
	externalMetrics := make(map[provider.ExternalMetricInfo]struct{})
	for _, cm := range fcp.externalMetrics {
		externalMetrics[provider.ExternalMetricInfo{
			Metric: cm,
		}] = struct{}{}
	}
	return externalMetrics, nil
}

func newFakeRegistry() *fakeRegistry {
	fakeClientProvider := &fakeMetricsClientsProvider{
		clients: make(map[string]*fakeMetricsClient),
	}
	return &fakeRegistry{
		registry: &Registry{
			lock:                         sync.RWMutex{},
			cachedMetricsSourcesBySource: make(map[string]cachedMetricSource),
			customMetrics:                make(map[provider.CustomMetricInfo]*cachedMetricSources),
			externalMetrics:              make(map[provider.ExternalMetricInfo]*cachedMetricSources),
			clientProvider:               fakeClientProvider,
		},
		fakeClientProvider: fakeClientProvider,
	}
}

func fakeCustomMetricList(names ...string) []provider.CustomMetricInfo {
	result := make([]provider.CustomMetricInfo, len(names))
	for i, name := range names {
		result[i] = provider.CustomMetricInfo{
			GroupResource: schema.GroupResource{},
			Namespaced:    false,
			Metric:        name,
		}
	}
	return result
}

func fakeExternalMetricList(names ...string) []provider.ExternalMetricInfo {
	result := make([]provider.ExternalMetricInfo, len(names))
	for i, name := range names {
		result[i] = provider.ExternalMetricInfo{
			Metric: name,
		}
	}
	return result
}

// - Expectations

type expectation struct {
	metricName         string
	metricType         v1alpha1.MetricType
	expectedSourceName string
	expectedError      func(err error) bool
}

func assertMetricsExpectations(t *testing.T, registry *Registry, expectations []expectation) {
	t.Helper()
	for _, expectation := range expectations {
		switch expectation.metricType {
		case v1alpha1.ExternalMetrics:
			assertExternalMetric(t, registry, expectation)
		case v1alpha1.CustomMetrics:
			assertCustomMetric(t, registry, expectation)
		default:
			t.Errorf("unknown metric type: \"%s\"", expectation.metricType)
		}
	}
}

func assertCustomMetric(t *testing.T, registry *Registry, expectated expectation) {
	t.Helper()
	metricInfo := provider.CustomMetricInfo{
		GroupResource: schema.GroupResource{},
		Namespaced:    false,
		Metric:        expectated.metricName,
	}
	backend, err := registry.GetMetricsBackend(metricInfo)
	if expectated.expectedError != nil && expectated.expectedError(err) {
		// This is an expected error
		return
	}
	if err != nil {
		t.Errorf("Registry.GetMetricsBackend() unexpected error = %v", err)
		return
	}
	// Check that the appropriate backend has been selected
	assert.NotNil(t, backend)
	fakeMetricsClient, ok := backend.(*fakeMetricsClient)
	if !ok {
		t.Errorf("fakeMetricsClient implementation expected")
	}
	assert.Equal(
		t, expectated.expectedSourceName, fakeMetricsClient.name,
		"metric %s was expected to be served by %s, but got %s as source", expectated.metricName, expectated.expectedSourceName, fakeMetricsClient.name,
	)
}

func assertExternalMetric(t *testing.T, registry *Registry, expectated expectation) {
	t.Helper()
	metricInfo := provider.ExternalMetricInfo{
		Metric: expectated.metricName,
	}
	backend, err := registry.GetExternalMetricsBackend(metricInfo)
	if expectated.expectedError != nil && expectated.expectedError(err) {
		// This is an expected error
		return
	}
	if err != nil {
		t.Errorf("Registry.GetExternalMetricsBackend() unexpected error = %v", err)
		return
	}
	// Check that the appropriate backend has been selected
	assert.NotNil(t, backend)
	fakeMetricsClient, ok := backend.(*fakeMetricsClient)
	if !ok {
		t.Errorf("fakeMetricsClient implementation expected")
	}
	assert.Equal(
		t, expectated.expectedSourceName, fakeMetricsClient.name,
		"metric %s was expected to be served by %s, but got %s as source", expectated.metricName, expectated.expectedSourceName, fakeMetricsClient.name,
	)
}
