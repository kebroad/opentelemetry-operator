// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconcile

import (
	"testing"
	"time"

	colfeaturegate "go.opentelemetry.io/collector/featuregate"

	"github.com/prometheus/prometheus/discovery/http"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

func TestPrometheusParser(t *testing.T) {
	param, err := newParams("test/test-img", "../testdata/http_sd_config_test.yaml")
	assert.NoError(t, err)

	t.Run("should update config with http_sd_config", func(t *testing.T) {
		actualConfig, err := ReplaceConfig(param.Instance)
		assert.NoError(t, err)

		// prepare
		var cfg Config
		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		promCfg, err := yaml.Marshal(promCfgMap)
		assert.NoError(t, err)

		err = yaml.UnmarshalStrict(promCfg, &cfg)
		assert.NoError(t, err)

		// test
		expectedMap := map[string]bool{
			"prometheus": false,
			"service-x":  false,
		}
		for _, scrapeConfig := range cfg.PromConfig.ScrapeConfigs {
			assert.Len(t, scrapeConfig.ServiceDiscoveryConfigs, 1)
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[0].Name(), "http")
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[0].(*http.SDConfig).URL, "http://test-targetallocator:80/jobs/"+scrapeConfig.JobName+"/targets?collector_id=$POD_NAME")
			expectedMap[scrapeConfig.JobName] = true
		}
		for k := range expectedMap {
			assert.True(t, expectedMap[k], k)
		}
		assert.True(t, cfg.TargetAllocConfig == nil)
	})

	t.Run("should update config with targetAllocator block", func(t *testing.T) {
		err := colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), true)
		param.Instance.Spec.TargetAllocator.Enabled = true
		assert.NoError(t, err)
		actualConfig, err := ReplaceConfig(param.Instance)
		assert.NoError(t, err)

		// prepare
		var cfg Config
		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		promCfg, err := yaml.Marshal(promCfgMap)
		assert.NoError(t, err)

		err = yaml.UnmarshalStrict(promCfg, &cfg)
		assert.NoError(t, err)

		// test
		assert.Len(t, cfg.PromConfig.ScrapeConfigs, 0)
		expectedTAConfig := &targetAllocator{
			Endpoint:    "http://test-targetallocator:80",
			Interval:    30 * time.Second,
			CollectorID: "${POD_NAME}",
		}
		assert.Equal(t, expectedTAConfig, cfg.TargetAllocConfig)
		err = colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), false)
		assert.NoError(t, err)
	})

	t.Run("should not update config with http_sd_config", func(t *testing.T) {
		param.Instance.Spec.TargetAllocator.Enabled = false
		actualConfig, err := ReplaceConfig(param.Instance)
		assert.NoError(t, err)

		// prepare
		var cfg Config
		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		promCfg, err := yaml.Marshal(promCfgMap)
		assert.NoError(t, err)

		err = yaml.UnmarshalStrict(promCfg, &cfg)
		assert.NoError(t, err)

		// test
		expectedMap := map[string]bool{
			"prometheus": false,
			"service-x":  false,
		}
		for _, scrapeConfig := range cfg.PromConfig.ScrapeConfigs {
			assert.Len(t, scrapeConfig.ServiceDiscoveryConfigs, 2)
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[0].Name(), "file")
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[1].Name(), "static")
			expectedMap[scrapeConfig.JobName] = true
		}
		for k := range expectedMap {
			assert.True(t, expectedMap[k], k)
		}
		assert.True(t, cfg.TargetAllocConfig == nil)
	})

}
