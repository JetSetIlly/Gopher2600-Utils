package auditors

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

type HighHue struct {
	vcs         *hardware.VCS
	frameCt     int
	usesHighHue bool
}

// Initialise implements the Audit interface
func (audit *HighHue) Initialise(vcs *hardware.VCS) error {
	audit.vcs = vcs
	audit.vcs.TV.AddPixelRenderer(audit)
	return nil
}

// Check implements the Audit interface
func (audit *HighHue) Check() error {
	if audit.frameCt > 60 {
		return CheckEnded
	}
	return nil
}

// Finalise implements the Audit interface
func (audit *HighHue) Finalise(_ *strings.Builder) error {
	if audit.usesHighHue {
		return fmt.Errorf("ROM uses colour-lum value of $Ex or $Fx")
	}
	return FinalisedOk
}

// NewFrame implements the television.PixelRenderer() interface
func (audit *HighHue) NewFrame(frameInfo television.FrameInfo) error {
	audit.frameCt++
	return nil
}

// NewScanline implements the television.PixelRenderer() interface
func (audit *HighHue) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements the television.PixelRenderer() interface
func (audit *HighHue) SetPixels(sig []signal.SignalAttributes, last int) error {
	if !audit.vcs.TV.GetFrameInfo().Stable {
		return nil
	}

	for i := 0; i <= last; i++ {
		if !sig[i].VBlank && sig[i].Color != signal.VideoBlack {
			hue := (uint8(sig[i].Color) & 0xf0) >> 4
			if hue == 0x0e || hue == 0x0f {
				audit.usesHighHue = true
				return nil
			}
		}
	}

	return nil
}

// Reset implements the television.PixelRenderer() interface
func (audit *HighHue) Reset() {
}

// EndRendering implements the television.PixelRenderer() interface
func (audit *HighHue) EndRendering() error {
	return nil
}
