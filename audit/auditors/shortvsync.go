package auditors

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
)

type shortVsync struct {
	vcs        *hardware.VCS
	frameCt    int
	shortVsync bool
}

// ID implements the Audit interface
func (audit *shortVsync) ID() string {
	return "ShortVsync"
}

// Initialise implements the Audit interface
func (audit *shortVsync) Initialise(vcs *hardware.VCS) error {
	audit.vcs = vcs
	audit.vcs.TV.AddFrameTrigger(audit)
	return nil
}

// Check implements the Audit interface
func (audit *shortVsync) Check() error {
	if audit.frameCt > 60 {
		return CheckEnded
	}
	return nil
}

// Finalise implements the Audit interface
func (audit *shortVsync) Finalise(_ *strings.Builder) error {
	if audit.shortVsync {
		return fmt.Errorf("ROM generates a VSYNC signal that is too short")
	}
	return FinalisedOk
}

// NewFrame implements the television.FrameTrigger() interface
func (audit *shortVsync) NewFrame(frameInfo frameinfo.Current) error {
	audit.frameCt++
	if frameInfo.Stable {
		audit.shortVsync = audit.shortVsync || frameInfo.VSYNCcount < 3
	}
	return nil
}
