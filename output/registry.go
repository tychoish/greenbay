package output

import (
	"sync"

	"github.com/tychoish/grip"
)

// ResultsFactory defines the signature used by constructor functions
// for implementations of the ResultsProducer interface.
type ResultsFactory func() ResultsProducer

type resultsRegistry struct {
	factories map[string]ResultsFactory
	mutex     sync.RWMutex
}

var registry *resultsRegistry

func init() {
	registry = &resultsRegistry{
		factories: make(map[string]ResultsFactory),
	}

	registry.add("gotest", func() ResultsProducer {
		return &GoTest{}
	})

	registry.add("result", func() ResultsProducer {
		return &Results{}
	})
}

func (r *resultsRegistry) add(name string, factory ResultsFactory) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, ok := r.factories[name]
	grip.AlertWhenf(ok, "overwriting existing factory named '%s'", name)

	r.factories[name] = factory
}

func (r *resultsRegistry) get(name string) (ResultsFactory, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	factory, ok := r.factories[name]

	grip.AlertWhenf(!ok, "factory named '%s' does not exist", name)

	return factory, ok
}

////////////////////////////////////////////////////////////////////////
//
// Public access methods for the global registry
//
////////////////////////////////////////////////////////////////////////

// GetResultsFactory provides a public mechanism for accessing
// constructors for result formats.
func GetResultsFactory(name string) (ResultsFactory, bool) {
	return registry.get(name)
}

// AddFactory provides a mechanism for adding additional results
// output to output registry.
func AddFactory(name string, factory ResultsFactory) {
	registry.add(name, factory)
}
