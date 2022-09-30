package stats

import (
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// ContainerStats contains the information about container stats.
type ContainerStats struct {
	Attributes      ContainerAttributes
	CPUStats        ContainerCpuStats
	MemoryStats     ContainerMemoryStats
	FileSystemStats ContainerFileSystemStats
}

type ContainerCpuStats struct {
	Timestamp int64
	// Cumulative CPU usage (sum across all cores) since object creation.
	UsageCoreNanoSeconds uint64
	// Total CPU usage (sum of all cores) averaged over the sample window.
	// The "core" unit can be interpreted as CPU core-nanoseconds per second.
	UsageNanoCores uint64
}

type ContainerMemoryStats struct {
	Timestamp int64
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
}

type ContainerFileSystemStats struct {
	Timestamp int64
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
}

type FilesystemIdentifier struct {
	// Mountpoint of a filesystem.
	Mountpoint string
}

type ContainerAttributes struct {
	Id          string
	Metadata    *runtime.ContainerMetadata
	Labels      map[string]string
	Annotations map[string]string
}
