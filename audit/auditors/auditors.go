package auditors

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware"
)

type Audit interface {
	ID() string
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

func NormaliseID(id string) string {
	return strings.ToUpper(id)
}

// Factory is used to create an auditor instance by name
var Factory map[string]func() Audit

const DefaultAuditor = "default"

var definitions []func() Audit = []func() Audit{
	func() Audit { return &coluxxCount{} },
	func() Audit { return &highHue{} },
	func() Audit { return &vsyncWithoutVblank{} },
}

// turn definitions into the Factory
func init() {
	Factory = make(map[string]func() Audit)
	for _, f := range definitions {
		aud := f()
		Factory[NormaliseID(aud.ID())] = f
	}
	Factory[DefaultAuditor] = definitions[0]
}
