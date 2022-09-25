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

package server

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
)

func (c *criService) containerMetrics(
	meta containerstore.Metadata,
	stats *types.Metric,
) (*runtime.ContainerStats, error) {
	var cs runtime.ContainerStats
	var usedBytes, inodesUsed uint64
	sn, err := c.snapshotStore.Get(meta.ID)
	// If snapshotstore doesn't have cached snapshot information
	// set WritableLayer usage to zero
	if err == nil {
		usedBytes = sn.Size
		inodesUsed = sn.Inodes
	}
	cs.WritableLayer = &runtime.FilesystemUsage{
		Timestamp: sn.Timestamp,
		FsId: &runtime.FilesystemIdentifier{
			Mountpoint: c.imageFSPath,
		},
		UsedBytes:       &runtime.UInt64Value{Value: usedBytes},
		InodesUsed:      &runtime.UInt64Value{Value: inodesUsed},
		Device:          "a",
		FsType:          "a",
		LimitBytes:      &runtime.UInt64Value{Value: 1},
		UsageBytes:      &runtime.UInt64Value{Value: 1},
		BaseUsage:       &runtime.UInt64Value{Value: 1},
		AvailableBytes:  &runtime.UInt64Value{Value: 1},
		HasInodes:       false,
		InodeCapacity:   &runtime.UInt64Value{Value: 1},
		InodesAvailable: &runtime.UInt64Value{Value: 1},
		ReadsCompleted:  &runtime.UInt64Value{Value: 1},
		ReadsMerged:     &runtime.UInt64Value{Value: 1},
		SectorsRead:     &runtime.UInt64Value{Value: 1},
		ReadTime:        &runtime.UInt64Value{Value: 1},
		WritesCompleted: &runtime.UInt64Value{Value: 1},
		WritesMerged:    &runtime.UInt64Value{Value: 1},
		SectorsWritten:  &runtime.UInt64Value{Value: 1},
		WriteTime:       &runtime.UInt64Value{Value: 1},
		IoInProgress:    &runtime.UInt64Value{Value: 1},
		IoTime:          &runtime.UInt64Value{Value: 1},
		WeightedIoTime:  &runtime.UInt64Value{Value: 1},
	}
	cs.Attributes = &runtime.ContainerAttributes{
		Id:          meta.ID,
		Metadata:    meta.Config.GetMetadata(),
		Labels:      meta.Config.GetLabels(),
		Annotations: meta.Config.GetAnnotations(),
	}

	if stats != nil {
		s, err := typeurl.UnmarshalAny(stats.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to extract container metrics: %w", err)
		}

		cpuStats, err := c.cpuContainerStats(meta.ID, false /* isSandbox */, s, protobuf.FromTimestamp(stats.Timestamp))
		if err != nil {
			return nil, fmt.Errorf("failed to obtain cpu stats: %w", err)
		}
		cs.Cpu = cpuStats

		memoryStats, err := c.memoryContainerStats(meta.ID, s, protobuf.FromTimestamp(stats.Timestamp))
		if err != nil {
			return nil, fmt.Errorf("failed to obtain memory stats: %w", err)
		}
		cs.Memory = memoryStats

		processStats, err := c.processContainerStats(meta.ID, s, protobuf.FromTimestamp(stats.Timestamp))
		if err != nil {
			return nil, fmt.Errorf("failed to obtain process stats: %w", err)
		}
		cs.Process = processStats

		metricStats, err := c.prometheusMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to obtain prometheus stats: %w", err)
		}
		cs.PrometheusMetric = metricStats
	}

	return &cs, nil
}

// getWorkingSet calculates workingset memory from cgroup memory stats.
// The caller should make sure memory is not nil.
// workingset = usage - total_inactive_file
func getWorkingSet(memory *v1.MemoryStat) uint64 {
	if memory.Usage == nil {
		return 0
	}
	var workingSet uint64
	if memory.TotalInactiveFile < memory.Usage.Usage {
		workingSet = memory.Usage.Usage - memory.TotalInactiveFile
	}
	return workingSet
}

// getWorkingSetV2 calculates workingset memory from cgroupv2 memory stats.
// The caller should make sure memory is not nil.
// workingset = usage - inactive_file
func getWorkingSetV2(memory *v2.MemoryStat) uint64 {
	var workingSet uint64
	if memory.InactiveFile < memory.Usage {
		workingSet = memory.Usage - memory.InactiveFile
	}
	return workingSet
}

func isMemoryUnlimited(v uint64) bool {
	// Size after which we consider memory to be "unlimited". This is not
	// MaxInt64 due to rounding by the kernel.
	// TODO: k8s or cadvisor should export this https://github.com/google/cadvisor/blob/2b6fbacac7598e0140b5bc8428e3bdd7d86cf5b9/metrics/prometheus.go#L1969-L1971
	const maxMemorySize = uint64(1 << 62)

	return v > maxMemorySize
}

// https://github.com/kubernetes/kubernetes/blob/b47f8263e18c7b13dba33fba23187e5e0477cdbd/pkg/kubelet/stats/helper.go#L68-L71
func getAvailableBytes(memory *v1.MemoryStat, workingSetBytes uint64) uint64 {
	// memory limit - working set bytes
	if !isMemoryUnlimited(memory.Usage.Limit) {
		return memory.Usage.Limit - workingSetBytes
	}
	return 0
}

func getAvailableBytesV2(memory *v2.MemoryStat, workingSetBytes uint64) uint64 {
	// memory limit (memory.max) for cgroupv2 - working set bytes
	if !isMemoryUnlimited(memory.UsageLimit) {
		return memory.UsageLimit - workingSetBytes
	}
	return 0
}

func (c *criService) cpuContainerStats(ID string, isSandbox bool, stats interface{}, timestamp time.Time) (*runtime.CpuUsage, error) {
	switch metrics := stats.(type) {
	case *v1.Metrics:
		if metrics.CPU != nil && metrics.CPU.Usage != nil {

			return &runtime.CpuUsage{
				Timestamp:                   timestamp.UnixNano(),
				UsageCoreNanoSeconds:        &runtime.UInt64Value{Value: metrics.CPU.Usage.Total},
				UsageNanoCores:              &runtime.UInt64Value{Value: 1},
				CpuCfsThrottledPeriodsTotal: &runtime.UInt64Value{Value: metrics.CPU.Throttling.ThrottledPeriods},
				CpuCfsThrottledSecondsTotal: &runtime.UInt64Value{Value: metrics.CPU.Throttling.ThrottledTime},
				CpuSystemSecondsTotal:       &runtime.UInt64Value{Value: metrics.CPU.Usage.Kernel},
				CpuUsageSecondsTotal:        &runtime.UInt64Value{Value: metrics.CPU.Usage.Total},
				CpuUserSecondsTotal:         &runtime.UInt64Value{Value: metrics.CPU.Usage.User},
				TasksState: &runtime.ContainerTasksState{
					SleepingTasks:        &runtime.UInt64Value{Value: 1},
					RunningTasks:         &runtime.UInt64Value{Value: 1},
					StoppedTasks:         &runtime.UInt64Value{Value: 1},
					UninterruptibleTasks: &runtime.UInt64Value{Value: 1},
					IowaitingTasks:       &runtime.UInt64Value{Value: 1},
				},
			}, nil
		}
	case *v2.Metrics:
		if metrics.CPU != nil {
			// convert to nano seconds
			usageCoreNanoSeconds := metrics.CPU.UsageUsec * 1000

			return &runtime.CpuUsage{
				Timestamp:                   timestamp.UnixNano(),
				UsageCoreNanoSeconds:        &runtime.UInt64Value{Value: usageCoreNanoSeconds},
				UsageNanoCores:              &runtime.UInt64Value{Value: 1},
				CpuCfsThrottledPeriodsTotal: &runtime.UInt64Value{Value: metrics.CPU.NrPeriods},
				CpuCfsThrottledSecondsTotal: &runtime.UInt64Value{Value: metrics.CPU.NrThrottled},
				CpuSystemSecondsTotal:       &runtime.UInt64Value{Value: metrics.CPU.SystemUsec},
				CpuUsageSecondsTotal:        &runtime.UInt64Value{Value: metrics.CPU.UsageUsec},
				CpuUserSecondsTotal:         &runtime.UInt64Value{Value: metrics.CPU.UserUsec},
				TasksState: &runtime.ContainerTasksState{
					SleepingTasks:        &runtime.UInt64Value{Value: 1},
					RunningTasks:         &runtime.UInt64Value{Value: 1},
					StoppedTasks:         &runtime.UInt64Value{Value: 1},
					UninterruptibleTasks: &runtime.UInt64Value{Value: 1},
					IowaitingTasks:       &runtime.UInt64Value{Value: 1},
				},
			}, nil
		}
	default:
		return nil, fmt.Errorf("unexpected metrics type: %v", metrics)
	}
	return nil, nil
}

func (c *criService) memoryContainerStats(ID string, stats interface{}, timestamp time.Time) (*runtime.MemoryUsage, error) {
	switch metrics := stats.(type) {
	case *v1.Metrics:
		if metrics.Memory != nil && metrics.Memory.Usage != nil {
			workingSetBytes := getWorkingSet(metrics.Memory)

			return &runtime.MemoryUsage{
				Timestamp: timestamp.UnixNano(),
				WorkingSetBytes: &runtime.UInt64Value{
					Value: workingSetBytes,
				},
				AvailableBytes:  &runtime.UInt64Value{Value: getAvailableBytes(metrics.Memory, workingSetBytes)},
				UsageBytes:      &runtime.UInt64Value{Value: metrics.Memory.Usage.Usage},
				RssBytes:        &runtime.UInt64Value{Value: metrics.Memory.TotalRSS},
				PageFaults:      &runtime.UInt64Value{Value: metrics.Memory.TotalPgFault},
				MajorPageFaults: &runtime.UInt64Value{Value: metrics.Memory.TotalPgMajFault},
				MemoryCache:     &runtime.UInt64Value{Value: metrics.Memory.Cache},
				MemoryFailcnt:   &runtime.UInt64Value{Value: metrics.Memory.Usage.Failcnt},
				MaxUsageBytes:   &runtime.UInt64Value{Value: metrics.Memory.Usage.Max},
			}, nil
		}
	case *v2.Metrics:
		if metrics.Memory != nil {
			workingSetBytes := getWorkingSetV2(metrics.Memory)

			return &runtime.MemoryUsage{
				Timestamp: timestamp.UnixNano(),
				WorkingSetBytes: &runtime.UInt64Value{
					Value: workingSetBytes,
				},
				AvailableBytes: &runtime.UInt64Value{Value: getAvailableBytesV2(metrics.Memory, workingSetBytes)},
				UsageBytes:     &runtime.UInt64Value{Value: metrics.Memory.Usage},
				// Use Anon memory for RSS as cAdvisor on cgroupv2
				// see https://github.com/google/cadvisor/blob/a9858972e75642c2b1914c8d5428e33e6392c08a/container/libcontainer/handler.go#L799
				RssBytes:        &runtime.UInt64Value{Value: metrics.Memory.Anon},
				PageFaults:      &runtime.UInt64Value{Value: metrics.Memory.Pgfault},
				MajorPageFaults: &runtime.UInt64Value{Value: metrics.Memory.Pgmajfault},
				MemoryCache:     &runtime.UInt64Value{Value: metrics.Memory.File},
			}, nil
		}
	default:
		return nil, fmt.Errorf("unexpected metrics type: %v", metrics)
	}
	return nil, nil
}

func (c *criService) processContainerStats(ID string, stats interface{}, timestamp time.Time) (*runtime.ContainerProcessUsage, error) {
	switch metrics := stats.(type) {
	case *v1.Metrics:
		if metrics.Pids != nil {
			return &runtime.ContainerProcessUsage{
				Timestamp:    timestamp.UnixNano(),
				ThreadsMax:   &runtime.UInt64Value{Value: metrics.Pids.Limit},
				ThreadsCount: &runtime.UInt64Value{Value: metrics.Pids.Current},
			}, nil
		}
	case *v2.Metrics:
		if metrics.Pids != nil {
			return &runtime.ContainerProcessUsage{
				Timestamp:    timestamp.UnixNano(),
				ThreadsMax:   &runtime.UInt64Value{Value: metrics.Pids.Limit},
				ThreadsCount: &runtime.UInt64Value{Value: metrics.Pids.Current},
			}, nil
		}
	default:
		return nil, fmt.Errorf("unexpected metrics type: %v", metrics)
	}
	return nil, nil
}

func (c *criService) prometheusMetrics() (*runtime.Metric, error) {
	var m runtime.Metric
	m.Label = &runtime.LabelPair{
		Name:  "danielye: prometheus dummy metric name",
		Value: "danielye: prometheus dummy metric value",
	}
	m.Gauge = &runtime.Gauge{
		Value: 1.0,
	}
	m.Type = runtime.MetricType_COUNTER
	m.TimestampMs = 17
	// var prometheus_label runtime.LabelPair
	// var label_array []runtime.LabelPair = &runtime.Lba
	// {&runtime.LabelPair{
	// 	Name:  []string{"danielye: prometheus dummy metric name",",1"},
	// 	Value: []string{"danielye: prometheus dummy metric value"},
	// }}
	// m.Label = label_array[]runtime.LabelPair{}
	// var label_name runtime.LabelPair
	return &m, nil

}
