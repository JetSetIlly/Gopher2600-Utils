package auditors

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware"
)

type Audit interface {
	Initialise(vcs *hardware.VCS) error
	Check() error
	Finalise() error
}

var (
	CheckEnded  = fmt.Errorf("check ended")
	FinalisedOk = fmt.Errorf("finalised okay")
)
