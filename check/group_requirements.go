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
