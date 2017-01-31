// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package config

import (
	"io"
	"os"
	"path"
)

// A FileResolver resolves references to files
type FileResolver interface {
	Resolve(file string) io.ReadCloser
}

// A RelativeResolver resolves files relative to the given paths
type RelativeResolver struct {
	paths []string
}

// NewRelativeResolver returns a file resolver relative to the given paths
func NewRelativeResolver(paths ...string) FileResolver {
	pathList := make([]string, len(paths))

	copy(pathList, paths)

	if appRoot := os.Getenv(EnvironmentPrefix() + _root); appRoot != "" {
		// add the app root
		pathList = append(pathList, appRoot)
	} else if cwd, err := os.Getwd(); err == nil {
		// add the current cwd
		pathList = append(pathList, cwd)
	}

	// add the exe dir
	pathList = append(pathList, path.Dir(os.Args[0]))

	return &RelativeResolver{
		paths: pathList,
	}
}

// Resolve finds a reader relative to the given resolver
func (rr RelativeResolver) Resolve(file string) io.ReadCloser {
	if path.IsAbs(file) {
		if reader, err := os.Open(file); err == nil {
			return reader
		}
	}

	// loop the paths
	for _, v := range rr.paths {
		fp := path.Join(v, file)
		if reader, err := os.Open(fp); err == nil {
			return reader
		}
	}

	return nil
}
