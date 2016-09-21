package check

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
	"github.com/mongodb/greenbay"
	"github.com/pkg/errors"
)

// Base is a type that all new checks should compose, and provides an
// implementation of most common amboy.Job and greenbay.Check methods.
type Base struct {
	TaskID        string        `bson:"name" json:"name" yaml:"name"`
	IsComplete    bool          `bson:"completed" json:"completed" yaml:"completed"`
	WasSuccessful bool          `bson:"passed" json:"passed" yaml:"passed"`
	JobType       amboy.JobType `bson:"job_type" json:"job_type" yaml:"job_type"`
	Errors        []error       `bson:"errors" json:"errors" yaml:"errors"`
	Message       string        `bson:"message" json:"message" yaml:"message"`
	TestSuites    []string      `bson:"suites" json:"suites" yaml:"suites"`
	dep           dependency.Manager
	mutex         sync.RWMutex
}

//////////////////////////////////////////////////////////////////////
//
// greenbay.Checker base methods implementation
//
//////////////////////////////////////////////////////////////////////

// SetID makes it possible to change the ID of an amboy.Job which is
// not settable in that interface, and is necessary for
// greenbay.Checker implementations owing to how these jobs are
// constructed from the greenbay config file.
func (b *Base) SetID(n string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.TaskID = n
}

// Output returns a consistent output format for greenbay.Checks,
// which may be useful for generating common output formats.
func (b *Base) Output() greenbay.CheckOutput {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return greenbay.CheckOutput{
		Name:      b.ID(),
		Check:     b.Type().Name,
		Suites:    b.Suites(),
		Completed: b.IsComplete,
		Passed:    b.WasSuccessful,
		Error:     b.Error().Error(),
		Message:   b.Message,
	}
}

func (b *Base) setState(result bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.WasSuccessful = result
}

func (b *Base) setMessage(m interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	switch msg := m.(type) {
	case string:
		b.Message = msg
	case []string:
		b.Message = strings.Join(msg, "\n")
	case error:
		b.Message = msg.Error()
	case int:
		b.Message = strconv.Itoa(msg)
	default:
		b.Message = fmt.Sprintf("%+v", msg)
	}
}

// Suites reports which suites the current check belongs to.
func (b *Base) Suites() []string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.TestSuites
}

// SetSuites allows callers, typically the configuration parser, to
// set the suites.
func (b *Base) SetSuites(suites []string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.TestSuites = suites
}

// Name returns the name of the *check* rather than the name of the
// task.
func (b *Base) Name() string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.JobType.Name
}

//////////////////////////////////////////////////////////////////////
//
// amboy.Job base methods implementation
//
//////////////////////////////////////////////////////////////////////

// ID returns the name of the job, and is a component of the amboy.Job
// interface.
func (b *Base) ID() string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.TaskID
}

// Completed returns true if the job has been marked completed, and is
// a component of the amboy.Job interface.
func (b *Base) Completed() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.IsComplete
}

// Type returns the amboy.JobType specification for this object, and
// is a component of the amboy.Job interface.
func (b *Base) Type() amboy.JobType {
	return b.JobType
}

// Dependency returns an amboy Job dependency interface object, and is
// a component of the amboy.Job interface.
func (b *Base) Dependency() dependency.Manager {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.dep
}

// SetDependency allows you to inject a different amboy.Job dependency
// object, and is a component of the amboy.Job interface.
func (b *Base) SetDependency(d dependency.Manager) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.dep = d
}

func (b *Base) markComplete() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.IsComplete = true
}

func (b *Base) addError(err error) {
	if err != nil {
		b.mutex.Lock()
		defer b.mutex.Unlock()

		b.Errors = append(b.Errors, err)
	}
}

func (b *Base) hasErrors() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return len(b.Errors) > 0
}

func (b *Base) Error() error {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if len(b.Errors) == 0 {
		return nil
	}

	var outputs []string

	for _, err := range b.Errors {
		outputs = append(outputs, fmt.Sprintf("%+v", err))
	}

	return errors.New(strings.Join(outputs, "\n"))
}
