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

package service

import (
	"time"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"
	"go.uber.org/fx/tracing"
	"go.uber.org/fx/ulog"

	"github.com/go-validator/validator"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (svc *serviceCore) setupLogging() error {
	cfg := svc.configProvider.Get("logging")
	if cfg.HasValue() {
		if err := svc.logConfig.Configure(cfg); err != nil {
			return errors.Wrap(err, "failed to initialize logging from config")
		}
	} else {
		// if no config - default to the regular one
		svc.logConfig = ulog.DefaultConfiguration()
	}

	logger, err := svc.logConfig.Build(zap.Hooks(ulog.Metrics(svc.metrics)))
	if err != nil {
		return errors.Wrap(err, "failed to build the logger")
	}

	// TODO(glib): SetLogger returns a deferral to clean up global log which is not used
	ulog.SetLogger(logger)

	return nil
}

func (svc *serviceCore) setupStandardConfig() error {
	if err := svc.configProvider.Get(config.Root).PopulateStruct(&svc.standardConfig); err != nil {
		return errors.Wrap(err, "unable to load standard configuration")
	}

	if errs := validator.Validate(svc.standardConfig); errs != nil {
		zap.L().Error("Invalid service configuration", zap.Error(errs))
		return errors.Wrap(errs, "service configuration failed validation")
	}
	return nil
}

func (svc *serviceCore) setupMetrics() {
	if svc.Metrics() == nil {
		if svc.metricsClient == nil {
			svc.metricsClient = metrics.NewClient()
		}
		svc.metrics, svc.statsReporter, svc.metricsCloser = svc.metricsClient.RootScope(svc)
		svc.metricsClient.Freeze()
	}
}

func (svc *serviceCore) setupRuntimeMetricsCollector() error {
	if svc.RuntimeMetricsCollector() != nil {
		return nil
	}

	var runtimeMetricsConfig metrics.RuntimeConfig
	err := svc.configProvider.Get("metrics.runtime").PopulateStruct(&runtimeMetricsConfig)
	if err != nil {
		return errors.Wrap(err, "unable to load runtime metrics configuration")
	}
	svc.runtimeCollector = metrics.StartCollectingRuntimeMetrics(
		svc.metrics.SubScope("runtime"), time.Second, runtimeMetricsConfig,
	)
	return nil
}

func (svc *serviceCore) setupTracer() error {
	if svc.Tracer() != nil {
		return nil
	}
	if err := svc.configProvider.Get("tracing").PopulateStruct(&svc.tracerConfig); err != nil {
		return errors.Wrap(err, "unable to load tracing configuration")
	}
	tracer, closer, err := tracing.InitGlobalTracer(
		&svc.tracerConfig,
		svc.standardConfig.Name,
		zap.L(),
		svc.metrics,
	)
	if err != nil {
		return errors.Wrap(err, "unable to initialize global tracer")
	}
	svc.tracer = tracer
	svc.tracerCloser = closer
	return nil
}

func (svc *serviceCore) setupObserver() {
	if svc.observer != nil {
		loadInstanceConfig(svc.configProvider, "service", svc.observer)

		if shc, ok := svc.observer.(SetContainerer); ok {
			shc.SetContainer(svc)
		}
	}
}

func (svc *serviceCore) setupAuthClient() {
	if svc.authClient != nil {
		return
	}
	svc.authClient = auth.Load(svc)
}
