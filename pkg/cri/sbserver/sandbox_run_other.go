//go:build !windows && !linux
// +build !windows,!linux

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
	"github.com/containerd/containerd"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// cleanupSandboxFiles unmount some sandbox files, we rely on the removal of sandbox root directory to
// remove these files. Unmount should *NOT* return error if the mount point is already unmounted.
func (c *criService) cleanupSandboxFiles(id string, config *runtime.PodSandboxConfig) error {
	return nil
}

// taskOpts generates task options for a (sandbox) container.
func (c *criService) taskOpts(runtimeType string) []containerd.NewTaskOpts {
	return []containerd.NewTaskOpts{}
}
