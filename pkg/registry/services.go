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
	"sort"
)

type cachedMetricSources []cachedMetricSource

func newMetricsSources() *cachedMetricSources {
	serviceList := make(cachedMetricSources, 0)
	return &serviceList
}

func (c cachedMetricSources) Len() int {
	return len(c)
}

func (c cachedMetricSources) Less(i, j int) bool {
	if c[i].priority > c[j].priority {
		return true
	}
	if c[i].priority < c[j].priority {
		return false
	}

	// sort alphabetically using the sourceName at last resort
	return c[i].sourceName < c[j].sourceName
}

func (c cachedMetricSources) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c *cachedMetricSources) addOrUpdateSource(metricSource cachedMetricSource) {
	found := -1
	for i, s := range *c {
		if s.sourceName == metricSource.sourceName {
			found = i
			break
		}
	}
	if found != -1 {
		(*c)[found] = metricSource
	} else {
		*c = append(*c, metricSource)
	}
	sort.Sort(c)
}

func (c *cachedMetricSources) removeSource(sourceName string) (empty bool) {
	found := -1
	for i, s := range *c {
		if s.sourceName == sourceName {
			found = i
		}
	}
	if found != -1 {
		*c = append((*c)[:found], (*c)[found+1:]...)
		sort.Sort(c)
	}
	return c.Len() == 0
}

func (c *cachedMetricSources) getBestMetricService() (*cachedMetricSource, error) {
	if c.Len() == 0 {
		return nil, fmt.Errorf("no metric backend for metric")
	}
	service := (*c)[0]
	return &service, nil
}
