/*
Copyright 2022.
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

package settings

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestControllerSettings_Load(t *testing.T) {
	env := getCompleteEnv()
	setupTestEnv(env)
	defer cleanupTestEnv(env)

	expectedCS := &ControllerSettings{
		DevicePluginImage:         env["DEVICE_PLUGIN_IMAGE"],
		DriverHabanaImageBasename: env["DRIVER_HABANA_IMAGE_BASENAME"],
		NodeMetricsImage:          env["NODE_METRICS_IMAGE"],
	}

	cs := &ControllerSettings{}

	assert.NoError(t, cs.Load())
	assert.EqualValues(t, expectedCS, cs)
}

func TestControllerSettings_Load_withRandomEnvVarMissing(t *testing.T) {
	env := getCompleteEnv()

	// All env vars are required, so unset one at random
	// and check if an error pops up
	var random string
	for k := range env {
		random = k
		break
	}

	delete(env, random)

	setupTestEnv(env)
	defer cleanupTestEnv(env)

	cs := &ControllerSettings{}
	err := cs.Load()

	expectedErrMessage := fmt.Sprintf("%s: environment variable is not set", random)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), expectedErrMessage)
	}
}

func TestControllerSettings_Load_withMultipleEnvVarsMissing(t *testing.T) {
	tests := []struct {
		missingEnvVars []string
	}{
		{missingEnvVars: []string{"DEVICE_PLUGIN_IMAGE"}},
		{missingEnvVars: []string{"DRIVER_HABANA_IMAGE_BASENAME"}},
		{missingEnvVars: []string{"NODE_METRICS_IMAGE"}},
		{
			missingEnvVars: []string{
				"DEVICE_PLUGIN_IMAGE",
				"DRIVER_HABANA_IMAGE_BASENAME",
				"NODE_METRICS_IMAGE",
			},
		},
	}

	for _, tc := range tests {
		env := getCompleteEnv()

		for _, k := range tc.missingEnvVars {
			delete(env, k)
		}

		setupTestEnv(env)

		cs := &ControllerSettings{}
		err := cs.Load()

		if assert.Error(t, err) {
			for _, k := range tc.missingEnvVars {
				expectedErrMessage := fmt.Sprintf("%s: environment variable is not set", k)
				assert.Contains(t, err.Error(), expectedErrMessage)
			}
		}

		cleanupTestEnv(env)
	}
}

func getCompleteEnv() map[string]string {
	return map[string]string{
		"DEVICE_PLUGIN_IMAGE":          "device plugin image",
		"DRIVER_HABANA_IMAGE_BASENAME": "driver habana image basename",
		"NODE_METRICS_IMAGE":           "node metrics image",
	}
}

func setupTestEnv(env map[string]string) {
	for k, v := range env {
		os.Setenv(k, v)
	}
}

func cleanupTestEnv(env map[string]string) {
	for k := range env {
		os.Unsetenv(k)
	}
}
