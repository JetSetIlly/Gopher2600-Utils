package auditors

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware"
)

type Audit interface {
	Initialise(vcs *hardware.VCS) error
	Check() error
	Finalise(msg *strings.Builder) error
}

// sentinal errors
var (
	// returned by Check() function
	CheckEnded = fmt.Errorf("check ended")

	// returned by Finalise() function
	FinalisedOk = fmt.Errorf("finalised okay")
)
