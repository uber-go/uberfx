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

package main

import (
	"log"

	"go.uber.org/fx/dig"
	"go.uber.org/fx/examples/dig/handlers"
	"go.uber.org/fx/examples/dig/hello"
	"go.uber.org/fx/modules/uhttp"
	"go.uber.org/fx/service"
)

func main() {
	// Polite sayer and HelloHandler are injected into the dependency graph
	// This can be done from any package, for this example it's done in main
	err := dig.InjectAll(
		hello.NewPoliteSayer,
		handlers.NewHandler,
	)
	if err != nil {
		panic(err)
	}

	svc, err := service.WithModules(
		uhttp.New(router, []uhttp.Filter{}),
	).Build()

	if err != nil {
		log.Fatal("Unable to initialize service", "error", err)
	}

	svc.Start()
}

func router(_ service.Host) []uhttp.RouteHandler {
	// when service calls us to provide a handler, we simply grab one from DIG
	var h *handlers.HelloHandler
	err := dig.Resolve(&h)
	if err != nil {
		panic(err)
	}

	return []uhttp.RouteHandler{
		uhttp.NewRouteHandler("/", h),
	}
}
