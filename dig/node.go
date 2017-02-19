package dig

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

type graphNode interface {
	// Return value of the object
	value(g *graph) (reflect.Value, error)

	// Other things that need to be present before this object can be created
	dependencies() []interface{}
}

type objNode struct {
	fmt.Stringer

	obj         interface{}
	objType     reflect.Type
	cached      bool
	cachedValue reflect.Value
}

// Return the earlier provided instance
func (n objNode) value(g *graph) (reflect.Value, error) {
	return reflect.ValueOf(n.obj), nil
}

func (n objNode) dependencies() []interface{} {
	return nil
}

func (n objNode) String() string {
	return fmt.Sprintf(
		"(object) obj: %v, deps: nil, cached: %v, cachedValue: %v",
		n.objType,
		n.cached,
		n.cachedValue,
	)
}

type funcNode struct {
	objNode
	fmt.Stringer

	constructor interface{}
	deps        []interface{}
}

// Call the function and return the result
func (n *funcNode) value(g *graph) (reflect.Value, error) {
	if n.cached {
		return n.cachedValue, nil
	}

	ct := reflect.TypeOf(n.constructor)

	// check that all the dependencies have nodes present in the graph
	// doesn't mean everything will go smoothly during resolve, but it
	// drastically increases the chances that we're not missing something
	for _, node := range g.nodes {
		for _, dep := range node.dependencies() {
			// check that the dependency is a registered objNode
			if _, ok := g.nodes[dep]; !ok {
				err := fmt.Errorf("%v dependency of type %v is not registered", ct, dep)
				return reflect.Zero(ct), err
			}
		}
	}

	args := make([]reflect.Value, ct.NumIn(), ct.NumIn())
	for idx := range args {
		arg := ct.In(idx)
		if node, ok := g.nodes[arg]; ok {
			v, err := node.value(g)
			if err != nil {
				return reflect.Zero(n.objType), errors.Wrap(err, "dependency resolution failed")
			}
			args[idx] = v
		}
	}

	cv := reflect.ValueOf(n.constructor)
	v := cv.Call(args)[0]
	n.cached = true
	n.cachedValue = v

	return v, nil
}

func (n funcNode) dependencies() []interface{} {
	return n.deps
}

func (n funcNode) String() string {
	return fmt.Sprintf(
		"(function) deps: %v, type: %v, constructor: %v, cached: %v, cachedValue: %v",
		n.deps,
		n.objType,
		n.constructor,
		n.cached,
		n.cachedValue,
	)
}