package check

import (
	"os/exec"

	"github.com/pkg/errors"
)

type packageChecker func(string) (bool, string, error)

// this is populated in init.go's init(), to avoid init() ordering
// effects. Only used during the init process, so we don't need locks
// for this.
var packageCheckerRegistry map[string]packageChecker

func packageCheckerFactory(args []string) packageChecker {
	return func(name string) (bool, string, error) {
		args = append(args, name)

		out, err := exec.Command(args[0], args[1]...).CombinedOutput()
		if err != nil {
			return false, string(out), errors.Errorf("%s package '%s' is not installed (%s)",
				args[0], name, err.Error())
		}

		return true, string(out), nil
	}
}
