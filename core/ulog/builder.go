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

package ulog

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"go.uber.org/fx/core/config"

	"github.com/uber-go/zap"
)

// Configuration for logging with uberfx
type Configuration struct {
	Level   string
	File    *FileConfiguration
	Stdout  bool
	Verbose bool

	prefixWithFileLine *bool `yaml:"prefix_with_fileline"`
	TextFormatter      *bool // use TextFormatter (default json)
}

// FileConfiguration describes the properties needed to log to a file
type FileConfiguration struct {
	Enabled   bool
	Directory string
	FileName  string
}

// LogBuilder is the struct containing logger
type LogBuilder struct {
	log       zap.Logger
	logConfig Configuration
	level     zap.Level
}

var _development = strings.Contains(config.GetEnvironment(), "development")

// NewBuilder creates an empty builder to building ulog
func NewBuilder(keyvals ...interface{}) *LogBuilder {
	return &LogBuilder{
		level: -1,
	}
}

// WithConfiguration sets up configuration for the log builder
func (lb *LogBuilder) WithConfiguration(logConfig Configuration) *LogBuilder {
	lb.logConfig = logConfig
	return lb
}

// SetLogger sets the logger for use
func (lb *LogBuilder) SetLogger(zap zap.Logger) *LogBuilder {
	lb.log = zap
	return lb
}

// SetLevel sets the logger level
func (lb *LogBuilder) SetLevel(level zap.Level) *LogBuilder {
	lb.level = level
	return lb
}

// Build the ulog logger for use
func (lb *LogBuilder) Build() Log {
	var log zap.Logger

	// When setLogger is called, we will always use logger that has been set
	if lb.log != nil {
		return &baselogger{
			log: lb.log,
		}
	}

	if _development {
		log = lb.devLogger()
	} else {
		log = lb.Configure()
	}
	if lb.level > 0 {
		log.SetLevel(lb.level)
	}
	return &baselogger{
		log: log,
	}
}

func (lb *LogBuilder) devLogger() zap.Logger {
	options := make([]zap.Option, 0, 3)
	options = append(options, zap.DebugLevel)
	return zap.New(zap.NewTextEncoder(), options...)
}

func (lb *LogBuilder) defaultLogger() zap.Logger {
	options := make([]zap.Option, 0, 3)
	options = append(options, zap.InfoLevel, zap.Output(zap.AddSync(os.Stdout)))
	return zap.New(zap.NewJSONEncoder(), options...)
}

// Configure the calling logger based on provided configuration
func (lb *LogBuilder) Configure() zap.Logger {
	lb.log = lb.defaultLogger()

	options := make([]zap.Option, 0, 3)
	if lb.logConfig.Verbose {
		options = append(options, zap.DebugLevel)
	} else {
		lb.log.With(zap.String("Level", lb.logConfig.Level)).Debug("setting log level")
		var lv zap.Level
		err := lv.UnmarshalText([]byte(lb.logConfig.Level))
		if err != nil {
			lb.log.With(zap.String("Level", lb.logConfig.Level)).Debug("cannot parse log level. setting to Debug as default")
			options = append(options, zap.DebugLevel)
		} else {
			options = append(options, lv)
		}
	}
	if lb.logConfig.File == nil || !lb.logConfig.File.Enabled {
		options = append(options, zap.Output(zap.AddSync(ioutil.Discard)))
	} else {
		options = append(options, zap.Output(lb.fileOutput(lb.logConfig.File, lb.logConfig.Stdout, lb.logConfig.Verbose)))
	}

	if lb.logConfig.TextFormatter != nil && *lb.logConfig.TextFormatter {
		return zap.New(zap.NewTextEncoder(), options...)
	}
	return zap.New(zap.NewJSONEncoder(), options...)
}

func (lb *LogBuilder) fileOutput(cfg *FileConfiguration, stdout bool, verbose bool) zap.WriteSyncer {
	fileLoc := path.Join(cfg.Directory, cfg.FileName)
	lb.log.Debug("adding log file output to zap")
	err := os.MkdirAll(cfg.Directory, os.FileMode(0755))
	if err != nil {
		lb.log.Fatal("failed to create log directory: ", zap.Error(err))
	}
	file, err := os.OpenFile(fileLoc, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
	if err != nil {
		lb.log.Fatal("failed to open log file for writing.", zap.Error(err))
	}
	lb.log.With(zap.String("filename", fileLoc)).Debug("logfile created successfully")
	if verbose || stdout {
		return zap.AddSync(io.MultiWriter(os.Stdout, file))
	}
	return zap.AddSync(file)
}
