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

package provider

import (
	"fmt"

	"github.com/barkbay/custom-metrics-router/pkg/controllers/metricsource"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type FullMetricsProvider interface {
	provider.CustomMetricsProvider
	provider.ExternalMetricsProvider
}

type routedMetricsProvider struct {
	registry *metricsource.Registry
}

func NewRoutedProvider(customMetricRoutes *metricsource.Registry) FullMetricsProvider {
	return &routedMetricsProvider{
		registry: customMetricRoutes,
	}
}

func (r routedMetricsProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	backend, err := r.registry.GetMetricsBackend(info)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics backend: %v", err)
	}
	return backend.GetMetricByName(name, info, metricSelector)
}

func (r routedMetricsProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	backend, err := r.registry.GetMetricsBackend(info)
	if err != nil {
		return nil, err
	}
	return backend.GetMetricBySelector(namespace, selector, info, metricSelector)
}

func (r routedMetricsProvider) ListAllMetrics() []provider.CustomMetricInfo {
	return r.registry.ListAllCustomMetrics()
}

func (r routedMetricsProvider) GetExternalMetric(namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	backend, err := r.registry.GetExternalMetricsBackend(info)
	if err != nil {
		return nil, err
	}
	return backend.GetExternalMetric(info.Metric, namespace, metricSelector)
}

func (r routedMetricsProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	return r.registry.ListAllExternalMetrics()
}
