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
	"testing"

	"github.com/barkbay/custom-metrics-router/pkg/api/v1alpha1"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRegistry_AddOrUpdateSource(t *testing.T) {
	type args struct {
		source v1alpha1.MetricsSource
	}
	tests := []struct {
		name     string
		registry *Registry
		args     args
		// metricsCount is the number of metrics added by the new source
		metricsCount int
		// expectations validates that we get a given metric from the expected source
		expectations []expectation
		// expectedCustomMetrics are the expected custom metrics listed once AddOrUpdateSource has been run
		expectedCustomMetrics []provider.CustomMetricInfo
		// expectedExternalMetrics are the expected external metrics listed once AddOrUpdateSource has been run
		expectedExternalMetrics []provider.ExternalMetricInfo
		wantErr                 bool
	}{
		{
			name: "Add a new source serving 2 existing custom metrics and 1 new custom metric",
			registry: newFakeRegistry().
				// Add 2 existing metric sources
				addCustomMetrics("source1", 100, "metric1", "metric2").
				addCustomMetrics("source2", 200, "metric2").
				// new Metrics sources with 3 metrics
				servedCustomMetrics("newSource", "metric1", "metric2", "metric3").
				registry,
			args: args{source: v1alpha1.MetricsSource{
				ObjectMeta: metav1.ObjectMeta{Name: "newSource"},
				Spec: v1alpha1.MetricsSourceSpec{
					Priority:              42,
					MetricTypes:           v1alpha1.MetricTypes{v1alpha1.CustomMetrics},
					MetricsServiceBackend: v1alpha1.MetricsServiceBackend{Name: "newSource"},
				},
			}},
			metricsCount:          3,
			expectedCustomMetrics: fakeCustomMetricList("metric1", "metric2", "metric3"),
			expectations: []expectation{
				{
					metricType:         v1alpha1.CustomMetrics,
					metricName:         "metric1",
					expectedSourceName: "source1", // metric1 is served by newSource and source1, the latter has the highest priority
				},
				{
					metricType:         v1alpha1.CustomMetrics,
					metricName:         "metric2", // metric2 is served by all the sources
					expectedSourceName: "source2", //  source2 has highest priority (200)
				},
				{
					metricType:         v1alpha1.CustomMetrics,
					metricName:         "metric3",
					expectedSourceName: "newSource", //  metric3 is only served by the new source
				},
			},
		},
		{
			name: "Add a new source serving 2 existing external metrics and 1 new external metric",
			registry: newFakeRegistry().
				// Add 2 existing metric sources
				addExternalMetrics("source1", 100, "metric1", "metric2").
				addExternalMetrics("source2", 200, "metric2").
				// new Metrics sources with 3 metrics
				servedExternalMetrics("newSource", "metric1", "metric2", "metric3").
				registry,
			args: args{source: v1alpha1.MetricsSource{
				ObjectMeta: metav1.ObjectMeta{Name: "newSource"},
				Spec: v1alpha1.MetricsSourceSpec{
					Priority:              42,
					MetricTypes:           v1alpha1.MetricTypes{v1alpha1.ExternalMetrics},
					MetricsServiceBackend: v1alpha1.MetricsServiceBackend{Name: "newSource"},
				},
			}},
			metricsCount:            3,
			expectedExternalMetrics: fakeExternalMetricList("metric1", "metric2", "metric3"),
			expectations: []expectation{
				{
					metricType:         v1alpha1.ExternalMetrics,
					metricName:         "metric1",
					expectedSourceName: "source1", // metric1 is served by newSource and source1, the latter has the highest priority
				},
				{
					metricType:         v1alpha1.ExternalMetrics,
					metricName:         "metric2", // metric2 is served by all the sources
					expectedSourceName: "source2", //  source2 has highest priority (200)
				},
				{
					metricType:         v1alpha1.ExternalMetrics,
					metricName:         "metric3",
					expectedSourceName: "newSource", //  metric3 is only served by the new source
				},
			},
		},
		{
			name: "Update an existing custom metrics source: remove some previously served metrics and increase priority",
			registry: newFakeRegistry().
				// Add 2 existing metric sources
				addCustomMetrics("source1", 100, "metric1", "metric2", "metric4").
				addCustomMetrics("source2", 200, "metric2", "metric3").
				// source1 now serves metric1 and metric3
				servedCustomMetrics("source1", "metric1", "metric3"). // metric2 and metric4 are no more served by source1
				registry,
			args: args{source: v1alpha1.MetricsSource{
				ObjectMeta: metav1.ObjectMeta{Name: "source1"},
				Spec: v1alpha1.MetricsSourceSpec{
					Priority:              300, // priority of source1 is increased
					MetricTypes:           v1alpha1.MetricTypes{v1alpha1.CustomMetrics},
					MetricsServiceBackend: v1alpha1.MetricsServiceBackend{Name: "source1"},
				},
			}},
			metricsCount:          2,
			expectedCustomMetrics: fakeCustomMetricList("metric1", "metric2", "metric3"),
			expectations: []expectation{
				{
					metricType:    v1alpha1.CustomMetrics,
					metricName:    "metric4", // no more served
					expectedError: errors.IsNotFound,
				},
				{
					metricType:         v1alpha1.CustomMetrics,
					metricName:         "metric3", // metric3 is served by all the sources
					expectedSourceName: "source1", //  but source1 has now the highest priority (300)
				},
				{
					metricType:         v1alpha1.CustomMetrics,
					metricName:         "metric2",
					expectedSourceName: "source2", // metric2 is now only served by source2
				},
				{
					metricType:         v1alpha1.CustomMetrics,
					metricName:         "metric1",
					expectedSourceName: "source1", // metric1 is still served by source1
				},
			},
		},
		{
			name: "Update an existing external metrics source: remove some previously served metrics and increase priority",
			registry: newFakeRegistry().
				// Add 2 existing metric sources
				addExternalMetrics("source1", 100, "metric1", "metric2", "metric4").
				addExternalMetrics("source2", 200, "metric2", "metric3").
				// source1 now serves metric1 and metric3
				servedExternalMetrics("source1", "metric1", "metric3"). // metric2 and metric4 are no more served by source1
				registry,
			args: args{source: v1alpha1.MetricsSource{
				ObjectMeta: metav1.ObjectMeta{Name: "source1"},
				Spec: v1alpha1.MetricsSourceSpec{
					Priority:              300, // priority of source1 is increased
					MetricTypes:           v1alpha1.MetricTypes{v1alpha1.ExternalMetrics},
					MetricsServiceBackend: v1alpha1.MetricsServiceBackend{Name: "source1"},
				},
			}},
			metricsCount:            2,
			expectedExternalMetrics: fakeExternalMetricList("metric1", "metric2", "metric3"),
			expectations: []expectation{
				{
					metricType:    v1alpha1.ExternalMetrics,
					metricName:    "metric4", // no more served
					expectedError: errors.IsNotFound,
				},
				{
					metricType:         v1alpha1.ExternalMetrics,
					metricName:         "metric3", // metric3 is served by all the sources
					expectedSourceName: "source1", //  but source1 has now the highest priority (300)
				},
				{
					metricType:         v1alpha1.ExternalMetrics,
					metricName:         "metric2",
					expectedSourceName: "source2", // metric2 is now only served by source2
				},
				{
					metricType:         v1alpha1.ExternalMetrics,
					metricName:         "metric1",
					expectedSourceName: "source1", // metric1 is still served by source1
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.registry.AddOrUpdateSource(tt.args.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("Registry.AddOrUpdateSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.metricsCount {
				t.Errorf("Registry.AddOrUpdateSource() = %v, metricsCount %v", got, tt.metricsCount)
			}
			assert.ElementsMatch(t, tt.registry.ListAllCustomMetrics(), tt.expectedCustomMetrics)
			assert.ElementsMatch(t, tt.registry.ListAllExternalMetrics(), tt.expectedExternalMetrics)
			assertMetricsExpectations(t, tt.registry, tt.expectations)
		})
	}
}

func TestRegistry_DeleteSource(t *testing.T) {
	type args struct {
		sourceName string
	}
	tests := []struct {
		name     string
		args     args
		registry *Registry
		// expectations validates that we get a given metric from the expected source
		expectations []expectation
		// expectedCustomMetrics are the expected custom metrics listed once AddOrUpdateSource has been run
		expectedCustomMetrics []provider.CustomMetricInfo
		// expectedExternalMetrics are the expected external metrics listed once AddOrUpdateSource has been run
		expectedExternalMetrics []provider.ExternalMetricInfo
	}{
		{
			name: "Remove registry",
			args: args{sourceName: "source1"}, // remove source1 as a source
			registry: newFakeRegistry().
				// Add 2 existing metric sources
				addCustomMetrics("source1", 100, "custom_metric1", "custom_metric2", "custom_metric3").
				addCustomMetrics("source2", 200, "custom_metric1", "custom_metric2").
				addExternalMetrics("source1", 100, "external_metric1", "external_metric2", "external_metric3").
				addExternalMetrics("source2", 200, "external_metric1", "external_metric2").
				registry,
			expectedCustomMetrics:   fakeCustomMetricList("custom_metric1", "custom_metric2"),
			expectedExternalMetrics: fakeExternalMetricList("external_metric1", "external_metric2"),
			expectations: []expectation{
				{
					metricType:    v1alpha1.CustomMetrics,
					metricName:    "custom_metric3", // no more served
					expectedError: errors.IsNotFound,
				},
				{
					metricType:    v1alpha1.ExternalMetrics,
					metricName:    "metric3", // no more served
					expectedError: errors.IsNotFound,
				},
				{
					metricType:         v1alpha1.CustomMetrics,
					metricName:         "custom_metric1", // metric3 is served by all the sources
					expectedSourceName: "source2",        //  but source1 has now the highest priority (300)
				},
				{
					metricType:         v1alpha1.CustomMetrics,
					metricName:         "custom_metric2",
					expectedSourceName: "source2", // metric2 is now only served by source2
				},
			},
		},
		{
			name: "Remove the last registry",
			args: args{sourceName: "source1"}, // remove source1 as a source
			registry: newFakeRegistry().
				// Add 2 existing metric sources
				addCustomMetrics("source1", 100, "metric1", "metric2", "metric3").
				registry,
			expectedCustomMetrics: fakeCustomMetricList(),
			expectations: []expectation{
				{
					metricType:    v1alpha1.CustomMetrics,
					metricName:    "metric3", // no more served
					expectedError: errors.IsNotFound,
				},
				{
					metricType:    v1alpha1.CustomMetrics,
					metricName:    "metric2", // no more served
					expectedError: errors.IsNotFound,
				},
				{
					metricType:    v1alpha1.CustomMetrics,
					metricName:    "metric1", // no more served
					expectedError: errors.IsNotFound,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.registry.DeleteSource(tt.args.sourceName)
			assert.ElementsMatch(t, tt.registry.ListAllCustomMetrics(), tt.expectedCustomMetrics)
			assert.ElementsMatch(t, tt.registry.ListAllExternalMetrics(), tt.expectedExternalMetrics)
			assertMetricsExpectations(t, tt.registry, tt.expectations)
		})
	}
}
