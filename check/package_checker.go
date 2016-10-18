package check

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

type packageChecker func(string) (bool, string, error)

// this is populated in init.go's init(), to avoid init() ordering
// effects. Only used during the init process, so we don't need locks
// for this.
var packageCheckerRegistry map[string]packageChecker

func packageCheckerFactory(args []string) packageChecker {
	return func(name string) (bool, string, error) {
		localArgs := append(args, name)

		out, err := exec.Command(localArgs[0], localArgs[1:]...).CombinedOutput()
		output := strings.Trim(string(out), "\n\t ")
		if err != nil {
			return false, output, errors.Errorf("%s package '%s' is not installed (%s)",
				localArgs[0], name, err.Error())
		}

		return true, output, nil
	}
}
