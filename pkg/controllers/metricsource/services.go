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
	"sort"

	"github.com/barkbay/custom-metrics-router/pkg/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

type MetricsServices []v1alpha1.MetricsSource

func (m MetricsServices) Len() int {
	return len(m)
}

func (m MetricsServices) Less(i, j int) bool {
	if m[i].Spec.Priority < m[j].Spec.Priority {
		return true
	}
	if m[i].Spec.Priority > m[j].Spec.Priority {
		return false
	}

	// sort alphabetically at last resort
	return m[i].Name < m[j].Name
}

func (m MetricsServices) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func NewMetricServiceList() *MetricsServices {
	serviceList := make(MetricsServices, 0)
	return &serviceList
}

func (m *MetricsServices) AddOrUpdateSource(metricSource v1alpha1.MetricsSource) {
	found := -1
	for i, s := range *m {
		if s.Name == metricSource.Name && s.Namespace == metricSource.Namespace {
			found = i
			break
		}
	}
	if found != -1 {
		(*m)[found] = metricSource
	} else {
		*m = append(*m, metricSource)
	}
	sort.Sort(m)
}

func (m *MetricsServices) RemoveSource(nn types.NamespacedName) bool {
	found := -1
	for i, s := range *m {
		if s.Name == nn.Name && s.Namespace == nn.Namespace {
			found = i
		}
	}
	if found != -1 {
		*m = append((*m)[:found], (*m)[found+1:]...)
		sort.Sort(m)
	}
	if m.Len() > 0 {
		return true
	}
	return false
}

func (m *MetricsServices) GetBestMetricService() (*v1alpha1.MetricsSource, error) {
	if m.Len() == 0 {
		return nil, fmt.Errorf("no metric backend for metric")
	}
	service := (*m)[0]
	return &service, nil
}
