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

package stats

import (
	"time"

	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// ContainerStats contains the information about container stats.
type ContainerStats struct {
	// Timestamp of when stats were collected
	Timestamp time.Time
	// Cumulative CPU usage (sum across all cores) since object creation.
	UsageCoreNanoSeconds uint64
	Attributes           ContainerAttributes
	CPUStats             ContainerCpuStats
	MemoryStats          ContainerMemoryStats
	ProcessStats         ContainerProcessStats
	NetworkStats         ContainerNetworkStats
	FileSystemStats      ContainerFileSystemStats
}

type ContainerCpuStats struct {
	// Cumulative CPU usage (sum across all cores) since object creation.
	UsageCoreNanoSeconds uint64
	// Total CPU usage (sum of all cores) averaged over the sample window.
	// The "core" unit can be interpreted as CPU core-nanoseconds per second.
	UsageNanoCores uint64
	// Number of elapsed enforcement period intervals
	PeriodsTotal uint64
	// Number of throttled period intervals
	CpuCfsThrottledPeriodsTotal uint64
	// Total time duration the container has been throttled
	CpuCfsThrottledSecondsTotal uint64
	// Cumulative system cpu time consumed in seconds
	CpuSystemSecondsTotal uint64
	// Cumulative cpu time consumed in seconds
	CpuUsageSecondsTotal uint64
	// Cumulative user cpu time consumed in seconds
	CpuUserSecondsTotal uint64
}

type ContainerMemoryStats struct {
	// The amount of working set memory in bytes.
	WorkingSetBytes uint64
	// Available memory for use. This is defined as the memory limit - workingSetBytes.
	AvailableBytes uint64
	// Total memory in use. This includes all memory regardless of when it was accessed.
	UsageBytes uint64
	// The amount of anonymous and swap cache memory (includes transparent hugepages).
	RssBytes uint64
	// Cumulative number of minor page faults.
	PageFaults uint64
	// Cumulative number of major page faults.
	MajorPageFaults uint64
	//Number of bytes of page cache memory
	MemoryCache uint64
	//Number of memory usage hits limits
	MemoryFailCnt uint64
	//Maximum memory usage recorded in bytes
	MaxUsageBytes uint64
}

type ContainerProcessStats struct {
	// Number of open file descriptors for the container
	FileDescriptorCount uint64
	// Number of open sockets for the container
	SocketCount uint64
	// Maximum number of threads allowed inside the container, infinity if value is zero
	MaxThreads uint64
	// Number of threads running inside the container
	ThreadsCount uint64
}

type ContainerFileSystemStats struct {
	// The unique identifier of the filesystem.
	FsId FilesystemIdentifier
	// UsedBytes represents the bytes used for images on the filesystem.
	// This may differ from the total bytes used on the filesystem and may not
	// equal CapacityBytes - AvailableBytes.
	UsedBytes uint64
	// InodesUsed represents the inodes used by the images.
	// This may not equal InodesCapacity - InodesAvailable because the underlying
	// filesystem may also be used for purposes other than storing images.
	InodesUsed uint64
	//The block device name associated with the filesystem.
	Device string
	//Type of the file system
	FsType string
	// Number of bytes that can be consumed by the container on this filesystem.
	LimitBytes uint64
	// Number of bytes that is consumed by the container on this filesystem.
	UsageBytes uint64
	// Base Usage that is consumed by the container's writable layer.
	BaseUsage uint64
	// Number of bytes available for non-root user.
	AvailableBytes uint64
	// HasInodes when true, indicates that Inodes info will be available.
	HasInodes bool
	// Number of inodes
	InodeCapacity uint64
	// number of available inodes
	InodesAvailable uint64
	// Number of reads completed
	// This is the total number of reads completed successfully.
	ReadsCompleted uint64
	// Number of reads merged
	// Reads and writes which are adjacent to each other may be merged for
	// efficiency.  Thus two 4K reads may become one 8K read before it is
	// ultimately handed to the disk, and so it will be counted (and queued)
	// as only one I/O.  This field lets you know how often this was done.
	ReadsMerged uint64
	// Number of sectors read
	// This is the total number of sectors read successfully.
	SectorsRead uint64
	// Number of milliseconds spent reading
	// This is the total number of milliseconds spent by all reads (as
	// measured from __make_request() to end_that_request_last()).
	ReadTime uint64
	// Number of writes completed
	// This is the total number of writes completed successfully.
	WritesCompleted uint64
	// Number of writes merged
	// See the description of reads merged.
	WritesMerged uint64
	// Number of sectors written
	// This is the total number of sectors written successfully.
	SectorsWritten uint64
	// Number of milliseconds spent writing
	// This is the total number of milliseconds spent by all writes (as
	// measured from __make_request() to end_that_request_last()).
	WriteTime uint64
	// Number of I/Os currently in progress
	// The only field that should go to zero. Incremented as requests are
	// given to appropriate struct request_queue and decremented as they finish.
	IoInProgress uint64
	// Number of milliseconds spent doing I/Os
	// This field increases so long as field 9 is nonzero.
	IoTime uint64
	// weighted number of milliseconds spent doing I/Os
	// This field is incremented at each I/O start, I/O completion, I/O
	// merge, or read of these stats by the number of I/Os in progress
	// (field 9) times the number of milliseconds spent doing I/O since the
	// last update of this field.  This can provide an easy measure of both
	// I/O completion time and the backlog that may be accumulating.
	WeightedIoTime uint64
}

type FilesystemIdentifier struct {
	// Mountpoint of a filesystem.
	Mountpoint string
}

type ContainerNetworkStats struct {
	// The name of the network interface.
	Name string
	// Cumulative count of bytes received.
	RxBytes uint64
	// Cumulative count of receive errors encountered.
	RxErrors uint64
	// Cumulative count of bytes transmitted.
	TxBytes uint64
	// Cumulative count of transmit errors encountered.
	TxErrors uint64
	// Cumulative count of packets dropped while receiving
	RxDropped uint64
	// Cumulative count of packets received
	RxPackets uint64
	// Cumulative count of packets dropped while transmitting
	TxDropped uint64
	// Cumulative count of packets transmitted
	TxPackets uint64
}

type ContainerAttributes struct {
	Id          string
	Metadata    *runtime.ContainerMetadata
	Labels      map[string]string
	Annotations map[string]string
}
