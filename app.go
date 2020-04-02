// Copyright (c) 2019 Uber Technologies, Inc.
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

package fx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/dig"
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/fx/internal/lifecycle"
	"go.uber.org/multierr"
)

// DefaultTimeout is the default timeout for starting or stopping an
// application. It can be configured with the StartTimeout and StopTimeout
// options.
const DefaultTimeout = 15 * time.Second

// An Option configures an App using the functional options paradigm
// popularized by Rob Pike. If you're unfamiliar with this style, see
// https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html.
type Option interface {
	fmt.Stringer

	apply(*App)
}

// Provide registers any number of constructor functions, teaching the
// application how to instantiate various types. The supplied constructor
// function(s) may depend on other types available in the application, must
// return one or more objects, and may return an error. For example:
//
//  // Constructs type *C, depends on *A and *B.
//  func(*A, *B) *C
//
//  // Constructs type *C, depends on *A and *B, and indicates failure by
//  // returning an error.
//  func(*A, *B) (*C, error)
//
//  // Constructs types *B and *C, depends on *A, and can fail.
//  func(*A) (*B, *C, error)
//
// The order in which constructors are provided doesn't matter, and passing
// multiple Provide options appends to the application's collection of
// constructors. Constructors are called only if one or more of their returned
// types are needed, and their results are cached for reuse (so instances of a
// type are effectively singletons within an application). Taken together,
// these properties make it perfectly reasonable to Provide a large number of
// constructors even if only a fraction of them are used.
//
// See the documentation of the In and Out types for advanced features,
// including optional parameters and named instances.
//
// Constructor functions should perform as little external interaction as
// possible, and should avoid spawning goroutines. Things like server listen
// loops, background timer loops, and background processing goroutines should
// instead be managed using Lifecycle callbacks.
func Provide(constructors ...interface{}) Option {
	return provideOption{
		Targets: constructors,
		Stack:   fxreflect.CallerStack(1, 0),
	}
}

type provideOption struct {
	Targets []interface{}
	Stack   fxreflect.Stack
}

func (o provideOption) apply(app *App) {
	for _, target := range o.Targets {
		app.provides = append(app.provides, provide{
			Target: target,
			Stack:  o.Stack,
		})
	}
}

func (o provideOption) String() string {
	items := make([]string, len(o.Targets))
	for i, c := range o.Targets {
		items[i] = fxreflect.FuncName(c)
	}
	return fmt.Sprintf("fx.Provide(%s)", strings.Join(items, ", "))
}

// Invoke registers functions that are executed eagerly on application start.
// Arguments for these invocations are built using the constructors registered
// by Provide. Passing multiple Invoke options appends the new invocations to
// the application's existing list.
//
// Unlike constructors, invocations are always executed, and they're always
// run in order. Invocations may have any number of returned values. If the
// final returned object is an error, it's assumed to be a success indicator.
// All other returned values are discarded.
//
// Typically, invoked functions take a handful of high-level objects (whose
// constructors depend on lower-level objects) and introduce them to each
// other. This kick-starts the application by forcing it to instantiate a
// variety of types.
//
// To see an invocation in use, read through the package-level example. For
// advanced features, including optional parameters and named instances, see
// the documentation of the In and Out types.
func Invoke(funcs ...interface{}) Option {
	return invokeOption{
		Targets: funcs,
		Stack:   fxreflect.CallerStack(1, 0),
	}
}

type invokeOption struct {
	Targets []interface{}
	Stack   fxreflect.Stack
}

func (o invokeOption) apply(app *App) {
	for _, target := range o.Targets {
		app.invokes = append(app.invokes, invoke{
			Target: target,
			Stack:  o.Stack,
		})
	}
}

func (o invokeOption) String() string {
	items := make([]string, len(o.Targets))
	for i, f := range o.Targets {
		items[i] = fxreflect.FuncName(f)
	}
	return fmt.Sprintf("fx.Invoke(%s)", strings.Join(items, ", "))
}

// Error registers any number of errors with the application to short-circuit
// startup. If more than one error is given, the errors are combined into a
// single error.
//
// Similar to invocations, errors are applied in order. All Provide and Invoke
// options registered before or after an Error option will not be applied.
func Error(errs ...error) Option {
	return errorOption(errs)
}

type errorOption []error

func (errs errorOption) apply(app *App) {
	app.err = multierr.Append(app.err, multierr.Combine(errs...))
}

func (errs errorOption) String() string {
	return fmt.Sprintf("fx.Error(%v)", multierr.Combine(errs...))
}

// Options converts a collection of Options into a single Option. This allows
// packages to bundle sophisticated functionality into easy-to-use Fx modules.
// For example, a logging package might export a simple option like this:
//
//  package logging
//
//  var Module = fx.Provide(func() *log.Logger {
//    return log.New(os.Stdout, "", 0)
//  })
//
// A shared all-in-one microservice package could then use Options to bundle
// logging with similar metrics, tracing, and gRPC modules:
//
//  package server
//
//  var Module = fx.Options(
//    logging.Module,
//    metrics.Module,
//    tracing.Module,
//    grpc.Module,
//  )
//
// Since this all-in-one module has a minimal API surface, it's easy to add
// new functionality to it without breaking existing users. Individual
// applications can take advantage of all this functionality with only one
// line of code:
//
//  app := fx.New(server.Module)
//
// Use this pattern sparingly, since it limits the user's ability to customize
// their application.
func Options(opts ...Option) Option {
	return optionGroup(opts)
}

type optionGroup []Option

func (og optionGroup) apply(app *App) {
	for _, opt := range og {
		opt.apply(app)
	}
}

func (og optionGroup) String() string {
	items := make([]string, len(og))
	for i, opt := range og {
		items[i] = fmt.Sprint(opt)
	}
	return fmt.Sprintf("fx.Options(%s)", strings.Join(items, ", "))
}

// StartTimeout changes the application's start timeout.
func StartTimeout(v time.Duration) Option {
	return startTimeoutOption(v)
}

type startTimeoutOption time.Duration

func (t startTimeoutOption) apply(app *App) {
	app.startTimeout = time.Duration(t)
}

func (t startTimeoutOption) String() string {
	return fmt.Sprintf("fx.StartTimeout(%v)", time.Duration(t))
}

// StopTimeout changes the application's stop timeout.
func StopTimeout(v time.Duration) Option {
	return stopTimeoutOption(v)
}

type stopTimeoutOption time.Duration

func (t stopTimeoutOption) apply(app *App) {
	app.stopTimeout = time.Duration(t)
}

func (t stopTimeoutOption) String() string {
	return fmt.Sprintf("fx.StopTimeout(%v)", time.Duration(t))
}

// Printer is the interface required by Fx's logging backend. It's implemented
// by most loggers, including the one bundled with the standard library.
type Printer interface {
	Printf(string, ...interface{})
}

// Logger redirects the application's log output to the provided printer.
func Logger(p Printer) Option {
	return loggerOption{p}
}

type loggerOption struct{ p Printer }

func (l loggerOption) apply(app *App) {
	app.logger = &fxlog.Logger{Printer: l.p}
	app.lifecycle = &lifecycleWrapper{lifecycle.New(app.logger)}
}

func (l loggerOption) String() string {
	return fmt.Sprintf("fx.Logger(%v)", l.p)
}

// NopLogger disables the application's log output. Note that this makes some
// failures difficult to debug, since no errors are printed to console.
var NopLogger = Logger(nopLogger{})

type nopLogger struct{}

func (l nopLogger) Printf(string, ...interface{}) {
	return
}

func (nopLogger) String() string { return "NopLogger" }

// An App is a modular application built around dependency injection. Most
// users will only need to use the New constructor and the all-in-one Run
// convenience method. In more unusual cases, users may need to use the Err,
// Start, Done, and Stop methods by hand instead of relying on Run.
//
// New creates and initializes an App. All applications begin with a
// constructor for the Lifecycle type already registered.
//
// In addition to that built-in functionality, users typically pass a handful
// of Provide options and one or more Invoke options. The Provide options
// teach the application how to instantiate a variety of types, and the Invoke
// options describe how to initialize the application.
//
// When created, the application immediately executes all the functions passed
// via Invoke options. To supply these functions with the parameters they
// need, the application looks for constructors that return the appropriate
// types; if constructors for any required types are missing or any
// invocations return an error, the application will fail to start (and Err
// will return a descriptive error message).
//
// Once all the invocations (and any required constructors) have been called,
// New returns and the application is ready to be started using Run or Start.
// On startup, it executes any OnStart hooks registered with its Lifecycle.
// OnStart hooks are executed one at a time, in order, and must all complete
// within a configurable deadline (by default, 15 seconds). For details on the
// order in which OnStart hooks are executed, see the documentation for the
// Start method.
//
// At this point, the application has successfully started up. If started via
// Run, it will continue operating until it receives a shutdown signal from
// Done (see the Done documentation for details); if started explicitly via
// Start, it will operate until the user calls Stop. On shutdown, OnStop hooks
// execute one at a time, in reverse order, and must all complete within a
// configurable deadline (again, 15 seconds by default).
type App struct {
	err          error
	container    *dig.Container
	lifecycle    *lifecycleWrapper
	provides     []provide
	lateProvides []provide
	invokes      []invoke
	logger       *fxlog.Logger
	startTimeout time.Duration
	stopTimeout  time.Duration
	errorHooks   []ErrorHandler

	donesMu sync.RWMutex
	dones   []chan os.Signal
}

// provide is a single constructor provided to Fx.
type provide struct {
	// Constructor provided to Fx. This may be an fx.Annotated.
	Target interface{}

	// Stack trace of where this provide was made.
	Stack fxreflect.Stack
}

// invoke is a single invocation request to Fx.
type invoke struct {
	// Function to invoke.
	Target interface{}

	// Stack trace of where this invoke was made.
	Stack fxreflect.Stack
}

// ErrorHandler handles Fx application startup errors.
type ErrorHandler interface {
	HandleError(error)
}

// ErrorHook registers error handlers that implement error handling functions.
// They are executed on invoke failures. Passing multiple ErrorHandlers appends
// the new handlers to the application's existing list.
func ErrorHook(funcs ...ErrorHandler) Option {
	return errorHookOption(funcs)
}

type errorHookOption []ErrorHandler

func (eho errorHookOption) apply(app *App) {
	app.errorHooks = append(app.errorHooks, eho...)
}

func (eho errorHookOption) String() string {
	items := make([]string, len(eho))
	for i, eh := range eho {
		items[i] = fmt.Sprint(eh)
	}
	return fmt.Sprintf("fx.ErrorHook(%v)", strings.Join(items, ", "))
}

type errorHandlerList []ErrorHandler

func (ehl errorHandlerList) HandleError(err error) {
	for _, eh := range ehl {
		eh.HandleError(err)
	}
}

// New creates and initializes an App, immediately executing any functions
// registered via Invoke options. See the documentation of the App struct for
// details on the application's initialization, startup, and shutdown logic.
func New(opts ...Option) *App {
	logger := fxlog.New()
	lc := &lifecycleWrapper{lifecycle.New(logger)}

	app := &App{
		container:    dig.New(dig.DeferAcyclicVerification()),
		lifecycle:    lc,
		logger:       logger,
		startTimeout: DefaultTimeout,
		stopTimeout:  DefaultTimeout,
	}

	for _, opt := range opts {
		opt.apply(app)
	}

	for _, p := range app.provides {
		app.provide(p)
	}

	frames := fxreflect.CallerStack(0, 0) // include New in the stack for default Provides
	app.provide(provide{
		Target: func() Lifecycle { return app.lifecycle },
		Stack:  frames,
	})
	app.provide(provide{Target: app.shutdowner, Stack: frames})
	app.provide(provide{Target: app.dotGraph, Stack: frames})

	if err := app.executeLateProvide(); err != nil {
		app.err = err

		if dig.CanVisualizeError(err) {
			var b bytes.Buffer
			dig.Visualize(app.container, &b, dig.VisualizeError(err))
			err = errorWithGraph{
				graph: b.String(),
				err:   err,
			}
		}
		errorHandlerList(app.errorHooks).HandleError(err)
	}

	if app.err != nil {
		app.logger.Printf("Error after options were applied: %v", app.err)
		return app
	}

	if err := app.executeInvokes(); err != nil {
		app.err = err

		if dig.CanVisualizeError(err) {
			var b bytes.Buffer
			dig.Visualize(app.container, &b, dig.VisualizeError(err))
			err = errorWithGraph{
				graph: b.String(),
				err:   err,
			}
		}
		errorHandlerList(app.errorHooks).HandleError(err)
	}
	return app
}

// DotGraph contains a DOT language visualization of the dependency graph in
// an Fx application. It is provided in the container by default at
// initialization. On failure to build the dependency graph, it is attached
// to the error and if possible, colorized to highlight the root cause of the
// failure.
type DotGraph string

type errWithGraph interface {
	Graph() DotGraph
}

type errorWithGraph struct {
	graph string
	err   error
}

func (err errorWithGraph) Graph() DotGraph {
	return DotGraph(err.graph)
}

func (err errorWithGraph) Error() string {
	return err.err.Error()
}

// VisualizeError returns the visualization of the error if available.
func VisualizeError(err error) (string, error) {
	if e, ok := err.(errWithGraph); ok && e.Graph() != "" {
		return string(e.Graph()), nil
	}
	return "", errors.New("unable to visualize error")
}

// Run starts the application, blocks on the signals channel, and then
// gracefully shuts the application down. It uses DefaultTimeout to set a
// deadline for application startup and shutdown, unless the user has
// configured different timeouts with the StartTimeout or StopTimeout options.
// It's designed to make typical applications simple to run.
//
// However, all of Run's functionality is implemented in terms of the exported
// Start, Done, and Stop methods. Applications with more specialized needs
// can use those methods directly instead of relying on Run.
func (app *App) Run() {
	app.run(app.Done())
}

// Err returns any error encountered during New's initialization. See the
// documentation of the New method for details, but typical errors include
// missing constructors, circular dependencies, constructor errors, and
// invocation errors.
//
// Most users won't need to use this method, since both Run and Start
// short-circuit if initialization failed.
func (app *App) Err() error {
	return app.err
}

// Start kicks off all long-running goroutines, like network servers or
// message queue consumers. It does this by interacting with the application's
// Lifecycle.
//
// By taking a dependency on the Lifecycle type, some of the user-supplied
// functions called during initialization may have registered start and stop
// hooks. Because initialization calls constructors serially and in dependency
// order, hooks are naturally registered in dependency order too.
//
// Start executes all OnStart hooks registered with the application's
// Lifecycle, one at a time and in order. This ensures that each constructor's
// start hooks aren't executed until all its dependencies' start hooks
// complete. If any of the start hooks return an error, Start short-circuits,
// calls Stop, and returns the inciting error.
//
// Note that Start short-circuits immediately if the New constructor
// encountered any errors in application initialization.
func (app *App) Start(ctx context.Context) error {
	return withTimeout(ctx, app.start)
}

// Stop gracefully stops the application. It executes any registered OnStop
// hooks in reverse order, so that each constructor's stop hooks are called
// before its dependencies' stop hooks.
//
// If the application didn't start cleanly, only hooks whose OnStart phase was
// called are executed. However, all those hooks are executed, even if some
// fail.
func (app *App) Stop(ctx context.Context) error {
	return withTimeout(ctx, app.lifecycle.Stop)
}

// Done returns a channel of signals to block on after starting the
// application. Applications listen for the SIGINT and SIGTERM signals; during
// development, users can send the application SIGTERM by pressing Ctrl-C in
// the same terminal as the running process.
//
// Alternatively, a signal can be broadcast to all done channels manually by
// using the Shutdown functionality (see the Shutdowner documentation for details).
func (app *App) Done() <-chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	app.donesMu.Lock()
	app.dones = append(app.dones, c)
	app.donesMu.Unlock()
	return c
}

// StartTimeout returns the configured startup timeout. Apps default to using
// DefaultTimeout, but users can configure this behavior using the
// StartTimeout option.
func (app *App) StartTimeout() time.Duration {
	return app.startTimeout
}

// StopTimeout returns the configured shutdown timeout. Apps default to using
// DefaultTimeout, but users can configure this behavior using the StopTimeout
// option.
func (app *App) StopTimeout() time.Duration {
	return app.stopTimeout
}

func (app *App) dotGraph() (DotGraph, error) {
	var b bytes.Buffer
	err := dig.Visualize(app.container, &b)
	return DotGraph(b.String()), err
}

func (app *App) provide(p provide) {
	constructor := p.Target
	if app.err != nil {
		return
	}

	if reflect.TypeOf(constructor).Kind() == reflect.Func {
		ft := reflect.ValueOf(constructor).Type()

		for i := 0; i < ft.NumOut(); i++ {
			t := ft.Out(i)
			if t == reflect.TypeOf((*Option)(nil)).Elem() {
				app.lateProvides = append(app.lateProvides, p)
				return
			}
		}
	}

	app.logger.PrintProvide(constructor)

	if _, ok := constructor.(Option); ok {
		app.err = fmt.Errorf("fx.Option should be passed to fx.New directly, "+
			"not to fx.Provide: fx.Provide received %v from:\n%+v",
			constructor, p.Stack)
		return
	}

	if ann, ok := constructor.(Annotated); ok {
		var opts []dig.ProvideOption
		switch {
		case len(ann.Group) > 0 && len(ann.Name) > 0:
			app.err = fmt.Errorf(
				"fx.Annotated may specify only one of Name or Group: received %v from:\n%+v",
				ann, p.Stack)
			return
		case len(ann.Name) > 0:
			opts = append(opts, dig.Name(ann.Name))
		case len(ann.Group) > 0:
			opts = append(opts, dig.Group(ann.Group))

		}

		if err := app.container.Provide(ann.Target, opts...); err != nil {
			app.err = fmt.Errorf("fx.Provide(%v) from:\n%+vFailed: %v", ann, p.Stack, err)
		}
		return
	}

	if reflect.TypeOf(constructor).Kind() == reflect.Func {
		ft := reflect.ValueOf(constructor).Type()

		for i := 0; i < ft.NumOut(); i++ {
			t := ft.Out(i)

			if t == reflect.TypeOf(Annotated{}) {
				app.err = fmt.Errorf(
					"fx.Annotated should be passed to fx.Provide directly, "+
						"it should not be returned by the constructor: "+
						"fx.Provide received %v from:\n%+v",
					fxreflect.FuncName(constructor), p.Stack)
				return
			}
		}
	}

	if err := app.container.Provide(constructor); err != nil {
		app.err = fmt.Errorf("fx.Provide(%v) from:\n%+vFailed: %v", fxreflect.FuncName(constructor), p.Stack, err)
	}
}

// Execute invokes in order supplied to New, returning the first error
// encountered.
func (app *App) executeInvokes() error {
	// TODO: consider taking a context to limit the time spent running invocations.

	for _, i := range app.invokes {
		fn := i.Target
		fname := fxreflect.FuncName(fn)
		app.logger.Printf("INVOKE\t\t%s", fname)

		var err error
		if _, ok := fn.(Option); ok {
			err = fmt.Errorf("fx.Option should be passed to fx.New directly, "+
				"not to fx.Invoke: fx.Invoke received %v from:\n%+v",
				fn, i.Stack)
		} else {
			err = app.container.Invoke(fn)
		}

		if err != nil {
			app.logger.Printf("fx.Invoke(%v) called from:\n%+vFailed: %v", fname, i.Stack, err)
			return err
		}
	}

	return nil
}

// Execute lateProvides, returning the first error
// encountered.
func (app *App) executeLateProvide() error {
	// TODO: consider taking a context to limit the time spent running invocations.

	for i := 0; i < len(app.lateProvides); i++ {
		p := app.lateProvides[i]

		fn := p.Target
		fname := fxreflect.FuncName(fn)
		app.logger.Printf("LATE PROVIDE\t\t%s", fname)

		if _, ok := fn.(Option); ok {
			return fmt.Errorf("fx.Option should be passed to fx.New directly, "+
				"not to fx.LateProvide: fx.LateProvide received %v from:\n%+v",
				fn, p.Stack)
		}

		invokeReturn, err := app.container.InvokeReturn(fn)
		if err != nil {
			app.logger.Printf("fx.LateProvide(%v) called from:\n%+vFailed: %v", fname, p.Stack, err)
			return err
		}

		for _, value := range invokeReturn {
			// The function returned another function
			if reflect.TypeOf(value).Kind() == reflect.Func {
				ft := reflect.ValueOf(value).Type()

				for a := 0; a < ft.NumOut(); a++ {
					t := ft.Out(a)
					if t == reflect.TypeOf((*Option)(nil)).Elem() {
						app.lateProvides = append(app.lateProvides, provide{
							Target: value,
							Stack:  fxreflect.CallerStack(0, 1),
						})
					}
				}
			} else if option, ok := value.(Option); ok {
				if p, ok := option.(provideOption); ok {
					for _, target := range p.Targets {
						app.lateProvides = append(app.lateProvides, provide{
							Target: target,
							Stack:  p.Stack,
						})
					}
				} else {
					option.apply(app)
				}
			}
		}
	}

	return nil
}

func (app *App) run(done <-chan os.Signal) {
	startCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout())
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		app.logger.Fatalf("ERROR\t\tFailed to start: %v", err)
	}

	app.logger.PrintSignal(<-done)

	stopCtx, cancel := context.WithTimeout(context.Background(), app.StopTimeout())
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {
		app.logger.Fatalf("ERROR\t\tFailed to stop cleanly: %v", err)
	}
}

func (app *App) start(ctx context.Context) error {
	if app.err != nil {
		// Some provides failed, short-circuit immediately.
		return app.err
	}

	// Attempt to start cleanly.
	if err := app.lifecycle.Start(ctx); err != nil {
		// Start failed, roll back.
		app.logger.Printf("ERROR\t\tStart failed, rolling back: %v", err)
		if stopErr := app.lifecycle.Stop(ctx); stopErr != nil {
			app.logger.Printf("ERROR\t\tCouldn't rollback cleanly: %v", stopErr)
			return multierr.Append(err, stopErr)
		}
		return err
	}

	app.logger.Printf("RUNNING")
	return nil
}

func withTimeout(ctx context.Context, f func(context.Context) error) error {
	c := make(chan error, 1)
	go func() { c <- f(ctx) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-c:
		return err
	}
}
