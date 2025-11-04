package auditors

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
)

type indeterminate struct {
	vcs     *hardware.VCS
	frameCt int
	hasLAX  bool
	hasXAA  bool
}

// ID implements the Audit interface
func (audit *indeterminate) ID() string {
	return "Indeterminate"
}

// Initialise implements the Audit interface
func (audit *indeterminate) Initialise(vcs *hardware.VCS) error {
	audit.vcs = vcs
	audit.vcs.TV.AddFrameTrigger(audit)
	return nil
}

// Check implements the Audit interface
func (audit *indeterminate) Check() error {
	if audit.frameCt > 60 {
		return CheckEnded
	}
	if audit.vcs.CPU.LastResult.Final {
		audit.hasLAX = audit.hasLAX || audit.vcs.CPU.LastResult.Defn.OpCode == 0xab
		audit.hasXAA = audit.hasXAA || audit.vcs.CPU.LastResult.Defn.Operator == 0x8b
	}
	return nil
}

// Finalise implements the Audit interface
func (audit *indeterminate) Finalise(_ *strings.Builder) error {
	if audit.hasLAX && audit.hasXAA {
		return fmt.Errorf("ROM uses both LAX (immediate) and XAA")
	}
	if audit.hasLAX {
		return fmt.Errorf("ROM uses LAX (immediate)")
	}
	if audit.hasXAA {
		return fmt.Errorf("ROM uses XAA")
	}
	return FinalisedOk
}

// NewFrame implements the television.PixelRenderer() interface
func (audit *indeterminate) NewFrame(frameInfo frameinfo.Current) error {
	audit.frameCt++
	return nil
}
