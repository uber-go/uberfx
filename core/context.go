// Copyright (c) 2016 Uber Technologies, Inc.
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

package core

import (
	gcontext "context"

	"go.uber.org/fx/service"
)

// Context embeds Host and go context for use
type Context interface {
	gcontext.Context
	service.Host
	WithContext(ctx gcontext.Context) *context
}

type context struct {
	gcontext.Context
	service.Host
}

// WithContext returns a shallow copy of c with its context changed to ctx.
// The provided ctx must be non-nil. Follows from net/http Request WithContext.
func (c *context) WithContext(ctx gcontext.Context) (newC *context) {
	if ctx == nil {
		panic("nil context")
	}
	newC = new(context)
	*newC = *c
	newC.Context = ctx
	return newC
}

var _ Context = &context{}

// NewContext always returns core.Context for use in the service
func NewContext(ctx gcontext.Context, host service.Host) Context {
	return &context{
		Context: ctx,
		Host:    host,
	}
}
