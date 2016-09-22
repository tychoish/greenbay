package registry

import (
	"fmt"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/dependency"
)

// JobInterchange provides a consistent way to describe and reliably
// serialize Job objects between different queue
// instances. Interchange is also used internally as part of JobGroup
// Job type.
type JobInterchange struct {
	Name       string                 `json:"name" bson:"_id" yaml:"name"`
	Type       string                 `json:"type" bson:"type" yaml:"type"`
	Version    int                    `json:"version" bson:"version" yaml:"version"`
	Priority   int                    `bson:"priority" json:"priority" yaml:"priority"`
	Completed  bool                   `bson:"completed" json:"completed" yaml:"completed"`
	Dispatched bool                   `bson:"dispatched" json:"dispatched" yaml:"dispatched"`
	Job        []byte                 `json:"job,omitempty" bson:"job,omitempty" yaml:"job,omitempty"`
	Dependency *DependencyInterchange `json:"dependency,omitempty" bson:"dependency,omitempty" yaml:"dependency,omitempty"`
}

// MakeJobInterchange changes a Job interface into a JobInterchange
// structure, for easier serialization.
func MakeJobInterchange(j amboy.Job) (*JobInterchange, error) {
	typeInfo := j.Type()

	dep, err := makeDependencyInterchange(typeInfo.Format, j.Dependency())
	if err != nil {
		return nil, err
	}

	data, err := j.Export()
	if err != nil {
		return nil, err
	}

	output := &JobInterchange{
		Name:       j.ID(),
		Type:       typeInfo.Name,
		Version:    typeInfo.Version,
		Priority:   j.Priority(),
		Completed:  j.Completed(),
		Job:        data,
		Dependency: dep,
	}

	return output, nil
}

// ConvertToJob reverses the process of ConvertToInterchange and
// converts the interchange format to a Job object using the types in
// the registry. Returns an error if the job type of the
// JobInterchange object isn't registered or the current version of
// the job produced by the registry is *not* the same as the version
// of the Job.
func ConvertToJob(j *JobInterchange) (amboy.Job, error) {
	factory, err := GetJobFactory(j.Type)
	if err != nil {
		return nil, err
	}

	job := factory()

	if job.Type().Version != j.Version {
		return nil, fmt.Errorf("job '%s' (version=%d) does not match the current version (%d) for the job type '%s'",
			j.Name, j.Version, job.Type().Version, j.Type)
	}

	dep, err := convertToDependency(job.Type().Format, j.Dependency)
	if err != nil {
		return nil, err
	}

	err = job.Import(j.Job)
	job.SetDependency(dep)
	if err != nil {
		return nil, err
	}

	job.SetPriority(j.Priority)

	return job, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// DependencyInterchange objects are a standard form for
// dependency.Manager objects. Amboy (should) only pass
// DependencyInterchange objects between processes, which have the
// type information in easy to access and index-able locations.
type DependencyInterchange struct {
	Type       string   `json:"type" bson:"type" yaml:"type"`
	Version    int      `json:"version" bson:"version" yaml:"version"`
	Edges      []string `bson:"edges" json:"edges" yaml:"edges"`
	Dependency []byte   `json:"dependency" bson:"dependency" yaml:"dependency"`
}

// MakeDependencyInterchange converts a dependency.Manager document to
// its DependencyInterchange format.
func makeDependencyInterchange(f amboy.Format, d dependency.Manager) (*DependencyInterchange, error) {
	typeInfo := d.Type()

	data, err := amboy.ConvertTo(f, d)
	if err != nil {
		return nil, err
	}

	output := &DependencyInterchange{
		Type:       typeInfo.Name,
		Version:    typeInfo.Version,
		Edges:      d.Edges(),
		Dependency: data,
	}

	return output, nil
}

// convertToDependency uses the registry to convert a
// DependencyInterchange object to the correct dependnecy.Manager
// type.
func convertToDependency(f amboy.Format, d *DependencyInterchange) (dependency.Manager, error) {
	factory, err := GetDependencyFactory(d.Type)
	if err != nil {
		return nil, err
	}

	dep := factory()

	if dep.Type().Version != d.Version {
		return nil, fmt.Errorf("dependency '%s' (version=%d) does not match the current version (%d) for the dependency type '%s'",
			d.Type, d.Version, dep.Type().Version, dep.Type().Name)
	}

	// this works, because we want to use all the data from the
	// interchange object, but want to use the type information
	// associated with the object that we produced with the
	// factory.
	err = amboy.ConvertFrom(f, d.Dependency, dep)
	if err != nil {
		return nil, err
	}

	return dep, nil
}
