package output

import "github.com/mongodb/amboy"

// ResultsProducer defines a common interface for generating results
// in different formats.
type ResultsProducer interface {
	Populate(amboy.Queue) error
	ToFile(string) error
	Print() error
}
