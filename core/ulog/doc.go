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

// Package ulog is the Logging package.
//
//  package ulog provides an API wrapper around the logging libraryzap (https://github.com/uber-go/zap).
//  ulog uses the builder pattern to instantiate the logger. With LogBuilder you can perform pre-initialization set up
// by injecting configuration, custom logger and log level prior to building the usable
//  ulog.Log object. ulog.Loginterface provides a few benifits -
// - Decouple services from the logger used undeaneath the framework.
// - Easy to use API for logging.
// - Easily swappable backend logger without changing the service.
//
//
//   package main
//
//   import "go.uber.org/fx/core/ulog"
//
//   func main() {
//     // Initialize logger object
//     logBuilder := ulog.Builder()
//
//     // Optional, configure logger with configuration preferred by your service
//     logConfig := ulog.Configuration{}
//     logBuilder := logBuilder.WithConfiguration(&logConfig)
//
//     // build ulog.Log from logBuilder
//     log := lobBuilder.Build()
//
//     // Use logger in your service
//     log.Info("Message describing loggging reason", "key", "value")
//   }
//
// Note that the log methods ( Info, Warn, Debug) take parameters as key value
// pairs (message, (key, value)...)
//
//
//  ulog.Configuration can be set up in one of two ways, either by initializing the struct,
// or describing necessary
//  logging configuration in the YAML and populating using config package.
//
// • Defining config structure:
//
//   loggingConfig := ulog.Configuration{
//     Stdout: true,
//   }
//
// • Configuration defined in YAML:
//
//   logging:
//     stdout: true
//     level: debug
//
// You can initialize your own zap.Logger implementation and inject into ulog.
// To configure and inject
//  zap.Logger, set up the logger prior to building
// the
//  ulog.Log object.
//
//   func setupMyZapLogger(zaplogger zap.Logger) ulog.Log {
//     return ulog.Builder().SetLogger(zaplogger).Build()
//   }
//
// • Current benchmarks
//
// Current performance benchmark data with ulog interface, ulog baselogger struct, and zap.Logger
//
// |-------------------------------------------|----------|-----------|-----------|------------|
// |BenchmarkUlogWithoutFields-8               |5000000   |226 ns/op  |48 B/op    |1 allocs/op |
// |BenchmarkUlogWithFieldsLogIFace-8          |2000000   |1026 ns/op |1052 B/op  |19 allocs/op|
// |BenchmarkUlogWithFieldsBaseLoggerStruct-8  |2000000   |912 ns/op  |795 B/op   |18 allocs/op|
// |BenchmarkUlogWithFieldsZapLogger-8         |3000000   |558 ns/op  |513 B/op   |1 allocs/op |
// |BenchmarkUlogLiteWithFields-8              |3000000   |466 ns/op  |297 B/op   |7 allocs/op |
//
//
//
package ulog
