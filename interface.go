package greenbay

import "github.com/mongodb/amboy"

type Checker interface {
	//
	SetID(string)
	Output() CheckOutput
	SetSuites([]string)
	Suites() []string

	// Name returns the name of the checker. Use ID(), in the
	// amboy.Job interface to get a unique identifer for the task.
	Name() string

	// Checker composes the amboy.Job interface.
	amboy.Job
}

// CheckOutput
type CheckOutput struct {
	Completed bool
	Passed    bool
	Message   string
	Errors    []string
}
