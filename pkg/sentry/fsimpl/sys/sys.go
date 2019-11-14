// Copyright 2019 The gVisor Authors.
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

// Package sys implements sysfs.
package sys

import (
	"bytes"
	"fmt"

	"gvisor.dev/gvisor/pkg/abi/linux"
	"gvisor.dev/gvisor/pkg/sentry/context"
	"gvisor.dev/gvisor/pkg/sentry/fsimpl/kernfs"
	"gvisor.dev/gvisor/pkg/sentry/kernel/auth"
	"gvisor.dev/gvisor/pkg/sentry/vfs"
	"gvisor.dev/gvisor/pkg/syserror"
)

// MaxCPUCores is the number of CPU cores reported by sysfs.
const MaxCPUCores uint = 8

// FilesystemType implements vfs.FilesystemType.
type FilesystemType struct{}

// filesystem implements vfs.FilesystemImpl.
type filesystem struct {
	kernfs.Filesystem
}

// GetFilesystem implements vfs.FilesystemType.GetFilesystem.
func (FilesystemType) GetFilesystem(ctx context.Context, vfsObj *vfs.VirtualFilesystem, creds *auth.Credentials, source string, opts vfs.GetFilesystemOptions) (*vfs.Filesystem, *vfs.Dentry, error) {
	fs := &filesystem{}
	fs.Filesystem.Init(vfsObj)
	defaultMode := linux.FileMode(01555)
	root := fs.newDir(creds, defaultMode, map[string]*kernfs.Dentry{
		"block": fs.newDir(creds, defaultMode, nil),
		"bus":   fs.newDir(creds, defaultMode, nil),
		"class": fs.newDir(creds, defaultMode, map[string]*kernfs.Dentry{
			"power_supply": fs.newDir(creds, defaultMode, nil),
		}),
		"dev": fs.newDir(creds, defaultMode, nil),
		"devices": fs.newDir(creds, defaultMode, map[string]*kernfs.Dentry{
			"system": fs.newDir(creds, defaultMode, map[string]*kernfs.Dentry{
				"cpu": fs.newDir(creds, defaultMode, map[string]*kernfs.Dentry{
					"online":   fs.newCPUFile(creds, MaxCPUCores),
					"possible": fs.newCPUFile(creds, MaxCPUCores),
					"present":  fs.newCPUFile(creds, MaxCPUCores),
				}),
			}),
		}),
		"firmware": fs.newDir(creds, defaultMode, nil),
		"fs":       fs.newDir(creds, defaultMode, nil),
		"kernel":   fs.newDir(creds, defaultMode, nil),
		"module":   fs.newDir(creds, defaultMode, nil),
		"power":    fs.newDir(creds, defaultMode, nil),
	})
	return fs.VFSFilesystem(), root.VFSDentry(), nil
}

// dir implements kernfs.Inode.
type dir struct {
	kernfs.InodeAttrs
	kernfs.InodeNoDynamicLookup
	kernfs.InodeNotSymlink
	kernfs.InodeDirectoryNoNewChildren

	kernfs.OrderedChildren
	dentry kernfs.Dentry
}

func (fs *filesystem) newDir(creds *auth.Credentials, mode linux.FileMode, contents map[string]*kernfs.Dentry) *kernfs.Dentry {
	d := &dir{}
	d.InodeAttrs.Init(creds, fs.NextIno(), linux.ModeDirectory|0755)
	d.OrderedChildren.Init(kernfs.OrderedChildrenOptions{})
	d.dentry.Init(d)

	d.IncLinks(d.OrderedChildren.Populate(&d.dentry, contents))

	return &d.dentry
}

// SetStat implements kernfs.Inode.SetStat.
func (d *dir) SetStat(fs *vfs.Filesystem, opts vfs.SetStatOptions) error {
	return syserror.EPERM
}

// Open implements kernfs.Inode.Open.
func (d *dir) Open(rp *vfs.ResolvingPath, vfsd *vfs.Dentry, flags uint32) (*vfs.FileDescription, error) {
	fd := &kernfs.GenericDirectoryFD{}
	fd.Init(rp.Mount(), vfsd, &d.OrderedChildren, flags)
	return fd.VFSFileDescription(), nil
}

// cpuFile implements kernfs.Inode.
type cpuFile struct {
	kernfs.DynamicBytesFile
	maxCores uint
}

// Generate implements vfs.DynamicBytesSource.Generate.
func (c *cpuFile) Generate(ctx context.Context, buf *bytes.Buffer) error {
	fmt.Fprintf(buf, "0-%d", c.maxCores-1)
	return nil
}

func (fs *filesystem) newCPUFile(creds *auth.Credentials, maxCores uint) *kernfs.Dentry {
	c := &cpuFile{maxCores: maxCores}
	c.DynamicBytesFile.Init(creds, fs.NextIno(), c)
	d := &kernfs.Dentry{}
	d.Init(c)
	return d
}
