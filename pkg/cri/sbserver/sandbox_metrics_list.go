/*
   Copyright The containerd Authors.
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

package sbserver

import (
	"context"
	"fmt"

	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// ListPodSandboxStats returns stats of all ready sandboxes.
func (c *criService) ListPodSandboxMetrics(
	ctx context.Context,
	r *runtime.ListPodSandboxMetricsRequest,
) (*runtime.ListPodSandboxMetricsResponse, error) {
	sandboxes := c.sandboxStore.List()

	podSandboxMetrics := new(runtime.ListPodSandboxMetricsResponse)
	for _, sandbox := range sandboxes {
		metrics, err := metricsForSandbox(sandbox)
		if err != nil { //nolint:staticcheck // Ignore SA4023 as some platforms always return nil (unimplemented metrics)
			return nil, fmt.Errorf("failed to obtain metrics for sandbox %q: %w", sandbox.ID, err)
		}

		sandboxMetrics, err := c.podSandboxMetrics(ctx, sandbox, metrics)
		if err != nil { //nolint:staticcheck // Ignore SA4023 as some platforms always return nil (unimplemented metrics)
			return nil, fmt.Errorf("failed to decode sandbox container metrics for sandbox %q: %w", sandbox.ID, err)
		}
		podSandboxMetrics.PodMetrics = append(podSandboxMetrics.PodMetrics, sandboxMetrics)
	}

	return podSandboxMetrics, nil
}
