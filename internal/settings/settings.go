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
	"errors"
	"fmt"
	"os"
)

const (
	DevicePluginImageEnvVar         = "DEVICE_PLUGIN_IMAGE"
	DriverHabanaImageBasenameEnvVar = "DRIVER_HABANA_IMAGE_BASENAME"
	NodeMetricsImageEnvVar          = "NODE_METRICS_IMAGE"
)

var (
	errEnvVarNotSet = errors.New("environment variable is not set")
)

var Settings = ControllerSettings{}

type ControllerSettings struct {
	DevicePluginImage         string
	DriverHabanaImageBasename string
	NodeMetricsImage          string
}

func (r *ControllerSettings) Load() error {
	errs := []error{}
	var found bool

	r.DevicePluginImage, found = os.LookupEnv(DevicePluginImageEnvVar)
	if !found {
		errs = append(errs, fmt.Errorf("%v: %w", DevicePluginImageEnvVar, errEnvVarNotSet))
	}

	r.DriverHabanaImageBasename, found = os.LookupEnv(DriverHabanaImageBasenameEnvVar)
	if !found {
		errs = append(errs, fmt.Errorf("%v: %w", DriverHabanaImageBasenameEnvVar, errEnvVarNotSet))
	}

	r.NodeMetricsImage, found = os.LookupEnv(NodeMetricsImageEnvVar)
	if !found {
		errs = append(errs, fmt.Errorf("%v: %w", NodeMetricsImageEnvVar, errEnvVarNotSet))
	}

	if len(errs) > 0 {
		return fmt.Errorf("the following errors were detected: %v", errs)
	}

	return nil
}
