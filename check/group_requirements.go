package check

import "github.com/pkg/errors"

type GroupRequirements struct {
	Any  bool
	One  bool
	None bool
	All  bool
	Name string
}

func (gr GroupRequirements) GetResults(passes, failures int) (bool, err) {
	if gr.All {
		if failures > 0 {
			return false, nil
		}
	} else if gr.One {
		if passes != 1 {
			return false, nil
		}
	} else if gr.Any {
		if passes == 0 {
			return false, nil
		}
	} else if gr.None {
		if passes > 0 {
			return false, nil
		}
	} else {
		return false, errors.Errorf("incorrectly configured group check for %s", gr.Name)
	}

	return true, nil
}

func (gr GroupRequirements) Validate() err {
	opts := []bool{gr.All, gr.Any, gr.One, gr.None}
	active := 0

	for _, opt := range opts {
		if opt {
			active++
		}
	}

	if active != 1 {
		return errors.Errorf("specified incorrect number of options for a '%s' check: "+
			"[all=%t, one=%t, any=%t, none=%t]", gr.Name,
			gr.All, gr.One, gr.Any, gr.None)
	}

	return nil
}
