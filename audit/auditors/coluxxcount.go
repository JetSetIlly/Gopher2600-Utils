package auditors

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

type coluxxCount struct {
	vcs     *hardware.VCS
	frameCt int

	colourCounts [2][128]int
}

// ID implements the Audit interface
func (audit *coluxxCount) ID() string {
	return "COLUxxCount"
}

// Initialise implements the Audit interface
func (audit *coluxxCount) Initialise(vcs *hardware.VCS) error {
	audit.vcs = vcs
	audit.vcs.TV.AddFrameTrigger(audit)
	return nil
}

// Check implements the Audit interface
func (audit *coluxxCount) Check() error {
	if audit.frameCt > 60 {
		return CheckEnded
	}

	if audit.vcs.Mem.LastCPUWrite {
		// if last cycle was write to a COLUxx register add one to the count for
		// the written colour value
		//
		// maintain two buckets. one for the a high LSB in the address and one
		// for a low LSB
		addr := audit.vcs.Mem.LastCPUAddressMapped
		if addr == cpubus.WriteAddressByRegister[cpubus.COLUPF] ||
			addr == cpubus.WriteAddressByRegister[cpubus.COLUBK] ||
			addr == cpubus.WriteAddressByRegister[cpubus.COLUP0] ||
			addr == cpubus.WriteAddressByRegister[cpubus.COLUP1] {

			data := audit.vcs.Mem.LastCPUData
			audit.colourCounts[data&0x01][data>>1]++
		}

	}
	return nil
}

// Finalise implements the Audit interface
func (audit *coluxxCount) Finalise(msg *strings.Builder) error {
	var summary [2]int

	// count number of buckets that have something in them
	for bucket := range summary {
		for _, ct := range audit.colourCounts[bucket] {
			if ct > 0 {
				summary[bucket]++
			}
		}
	}

	msg.WriteString(fmt.Sprintf("| % 4d | % 4d", summary[0], summary[1]))

	// we'll always say the audit went okay in this case
	return FinalisedOk
}

// NewFrame implements the television.FrameTrigger() interface
func (audit *coluxxCount) NewFrame(frameInfo television.FrameInfo) error {
	audit.frameCt++
	return nil
}
