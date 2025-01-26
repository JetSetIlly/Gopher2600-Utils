package auditors

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

type VsyncWithoutVblank struct {
	vcs        *hardware.VCS
	frameCt    int
	usesVBLANK bool
}

// Initialise implements the Audit interface
func (audit *VsyncWithoutVblank) Initialise(vcs *hardware.VCS) error {
	audit.vcs = vcs
	audit.vcs.TV.AddFrameTrigger(audit)
	return nil
}

// Check implements the Audit interface
func (audit *VsyncWithoutVblank) Check() error {
	if audit.frameCt > 60 {
		return CheckEnded
	}

	if !audit.vcs.TV.GetFrameInfo().Stable {
		return nil
	}

	sig := audit.vcs.TV.GetLastSignal()
	audit.usesVBLANK = sig.VSync && sig.VBlank
	return nil
}

// Finalise implements the Audit interface
func (audit *VsyncWithoutVblank) Finalise(_ *strings.Builder) error {
	if !audit.usesVBLANK {
		return fmt.Errorf("ROM uses VSYNC without VBLANK")
	}
	return FinalisedOk
}

// NewFrame implements the television.FrameTrigger() interface
func (audit *VsyncWithoutVblank) NewFrame(frameInfo television.FrameInfo) error {
	audit.frameCt++
	return nil
}
