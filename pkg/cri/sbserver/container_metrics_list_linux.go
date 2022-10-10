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
	"fmt"
	"time"

	"github.com/containerd/containerd/api/types"
	v1 "github.com/containerd/containerd/metrics/types/v1"
	v2 "github.com/containerd/containerd/metrics/types/v2"
	"github.com/containerd/containerd/protobuf"
	"github.com/containerd/typeurl"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"

	containerstore "github.com/containerd/containerd/pkg/cri/store/container"
	"github.com/containerd/containerd/pkg/cri/store/stats"
	containerstorestats "github.com/containerd/containerd/pkg/cri/store/stats"
)

func (c *criService) containerUnstructuredMetrics(
	meta containerstore.Metadata,
	stats *types.Metric,
) (*runtime.ContainerStats, error) {
	var cs runtime.ContainerStats
	generatedMetrics, err := c.generatedContainerMetrics(meta, stats)
	// If snapshotstore doesn't have cached snapshot information
	// set WritableLayer usage to zero
	if err != nil {
		return nil, fmt.Errorf("failed to extract container metrics: %w", err)
	}

	cs.Attributes = &runtime.ContainerAttributes{
		Id:          meta.ID,
		Metadata:    meta.Config.GetMetadata(),
		Labels:      meta.Config.GetLabels(),
		Annotations: meta.Config.GetAnnotations(),
	}
	cs.Cpu = &runtime.CpuUsage{
		Timestamp:            generatedMetrics.ContainerCPUStats.Timestamp,
		UsageCoreNanoSeconds: &runtime.UInt64Value{Value: generatedMetrics.ContainerCPUStats.UsageCoreNanoSeconds},
		UsageNanoCores:       &runtime.UInt64Value{Value: generatedMetrics.ContainerCPUStats.UsageNanoCores},
	}
	cs.Memory = &runtime.MemoryUsage{
		Timestamp:       generatedMetrics.ContainerMemoryStats.Timestamp,
		WorkingSetBytes: &runtime.UInt64Value{Value: generatedMetrics.ContainerMemoryStats.WorkingSetBytes},
		AvailableBytes:  &runtime.UInt64Value{Value: generatedMetrics.ContainerMemoryStats.AvailableBytes},
		UsageBytes:      &runtime.UInt64Value{Value: generatedMetrics.ContainerMemoryStats.UsageBytes},
		RssBytes:        &runtime.UInt64Value{Value: generatedMetrics.ContainerMemoryStats.RssBytes},
		PageFaults:      &runtime.UInt64Value{Value: generatedMetrics.ContainerMemoryStats.PageFaults},
		MajorPageFaults: &runtime.UInt64Value{Value: generatedMetrics.ContainerMemoryStats.MajorPageFaults},
	}
	cs.WritableLayer = &runtime.FilesystemUsage{
		Timestamp:  generatedMetrics.ContainerFileSystemStats.Timestamp,
		FsId:       &runtime.FilesystemIdentifier{Mountpoint: generatedMetrics.ContainerFileSystemStats.FsID.Mountpoint},
		UsedBytes:  &runtime.UInt64Value{Value: generatedMetrics.ContainerFileSystemStats.UsedBytes},
		InodesUsed: &runtime.UInt64Value{Value: generatedMetrics.ContainerFileSystemStats.InodesUsed},
	}

	return &cs, nil
}

func (c *criService) generatedUnstructuredContainerMetrics(
	meta containerstore.Metadata,
	stats *types.Metric,
) (*containerstorestats.ContainerStats, error) {
	var cs containerstorestats.ContainerStats
	var usedBytes, inodesUsed uint64
	sn, err := c.snapshotStore.Get(meta.ID)
	// If snapshotstore doesn't have cached snapshot information
	// set WritableLayer usage to zero
	if err == nil {
		usedBytes = sn.Size
		inodesUsed = sn.Inodes
	}
	cs.ContainerFileSystemStats = containerstorestats.ContainerFileSystemStats{
		FsID: containerstorestats.FilesystemIdentifier{
			Mountpoint: c.imageFSPath,
		},
		UsedBytes:  usedBytes,
		InodesUsed: inodesUsed,
	}

	if stats != nil {
		s, err := typeurl.UnmarshalAny(stats.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to extract container metrics: %w", err)
		}

		err = c.setContainerStats(meta.ID, false, cs)
		if err != nil {
			return nil, fmt.Errorf("failed to set container stats: %w", err)
		}

		cpuStats, err := c.generatedCPUContainerStats(meta.ID, false /* isSandbox */, s, protobuf.FromTimestamp(stats.Timestamp))
		if err != nil {
			return nil, fmt.Errorf("failed to obtain cpu stats: %w", err)
		}
		cs.ContainerCPUStats = *cpuStats

		memoryStats, err := c.generatedMemoryContainerStats(meta.ID, s, protobuf.FromTimestamp(stats.Timestamp))
		if err != nil {
			return nil, fmt.Errorf("failed to obtain memory stats: %w", err)
		}
		cs.ContainerMemoryStats = *memoryStats
	}
	return &cs, nil
}

func (c *criService) generatedCPUContainerMetrics(ID string, isSandbox bool, stats interface{}, timestamp time.Time) (*containerstorestats.ContainerCPUMetrics, error) {
	switch metrics := stats.(type) {
	case *v1.Metrics:
		if metrics.CPU != nil && metrics.CPU.Usage != nil {
			return &containerstorestats.ContainerCPUMetrics{
				CpuCfsThrottledPeriodsTotal: metrics.CPU.Throttling.ThrottledPeriods,
				CpuCfsThrottledSecondsTotal: metrics.CPU.Throttling.ThrottledTime / uint64(time.Second),
				CpuSystemSecondsTotal:       metrics.CPU.Usage.Kernel,
				CpuUsageSecondsTotal:        metrics.CPU.Usage.Total,
				CpuUserSecondsTotal:         metrics.CPU.Usage.User,
			}, nil
		}
	case *v2.Metrics:
		if metrics.CPU != nil {
			return &containerstorestats.ContainerCPUMetrics{
				CpuCfsThrottledPeriodsTotal: metrics.CPU.NrPeriods,
				CpuCfsThrottledSecondsTotal: metrics.CPU.NrThrottled / uint64(time.Second),
				CpuSystemSecondsTotal:       metrics.CPU.SystemUsec,
				CpuUsageSecondsTotal:        metrics.CPU.UsageUsec,
				CpuUserSecondsTotal:         metrics.CPU.UserUsec,
			}, nil
		}
	default:
		return nil, fmt.Errorf("unexpected metrics type: %v", metrics)
	}
	return nil, nil
}

func (c *criService) generatedMemoryContainerMetrics(ID string, stats interface{}, timestamp time.Time) (*containerstorestats.ContainerMemoryMetrics, error) {
	switch metrics := stats.(type) {
	case *v1.Metrics:
		if metrics.Memory != nil && metrics.Memory.Usage != nil {
			return &containerstorestats.ContainerMemoryMetrics{
				MemoryCache:   metrics.Memory.Cache,
				MemoryFailCnt: metrics.Memory.Usage.Failcnt,
				MaxUsageBytes: metrics.Memory.Usage.Max,
			}, nil
		}
	case *v2.Metrics:
		if metrics.Memory != nil {
			return &containerstorestats.ContainerMemoryStats{
				MemoryCache: metrics.Memory.Cache,
			}, nil
		}
	default:
		return nil, fmt.Errorf("unexpected metrics type: %v", metrics)
	}
	return nil, nil
}

func (c *criService) setContainerStats(containerID string, isSandbox bool, cs containerstorestats.ContainerStats) error {
	var oldStats *stats.ContainerStats

	if isSandbox {
		sandbox, err := c.sandboxStore.Get(containerID)
		if err != nil {
			return fmt.Errorf("failed to get sandbox container: %s: %w", containerID, err)
		}
		oldStats = sandbox.Stats
	} else {
		container, err := c.containerStore.Get(containerID)
		if err != nil {
			return fmt.Errorf("failed to get container ID: %s: %w", containerID, err)
		}
		oldStats = container.Stats
	}

	if oldStats == nil {
		if isSandbox {
			err := c.sandboxStore.UpdateContainerStats(containerID, cs)
			if err != nil {
				return fmt.Errorf("failed to update sandbox stats container ID: %s: %w", containerID, err)
			}
		} else {
			err := c.containerStore.UpdateContainerStats(containerID, cs)
			if err != nil {
				return fmt.Errorf("failed to update container stats ID: %s: %w", containerID, err)
			}
		}
		return nil
	}

	return nil
}
