package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

var timedOut = fmt.Errorf("timed out")

func main() {
	// we don't want date/time in log entries
	log.SetFlags(0)

	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <path to ROMs>\n", os.Args[0])
	}

	// check path to roms argument
	f, err := os.Open(os.Args[1])
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}

	// new television with auto-selecting tv protocl
	tv, err := television.NewTelevision("AUTO")
	if err != nil {
		log.Fatal(err)
	}
	defer tv.End()
	tv.SetFPSCap(false)

	mix := &myAudioMixer{}
	tv.AddAudioMixer(mix)

	// new VCS
	vcs, err := hardware.NewVCS(tv, nil)
	if err != nil {
		log.Fatal(err)
	}

	// table header
	fmt.Printf("%s-+-%s-+-%s-+-%s-+-%v\n",
		strings.Repeat("-", 30), strings.Repeat("-", 6),
		strings.Repeat("-", 4), strings.Repeat("-", 10),
		strings.Repeat("-", 30))
	fmt.Printf("%30s | %6s | %4s | %10s | %v\n", "Name", "Format", "TV", "Audio", "Errors")
	fmt.Printf("%30s | %6s | %4s | %10s | %v\n", "", "", "", "[at start]", "")
	fmt.Printf("%s-+-%s-+-%s-+-%s-+-%v\n",
		strings.Repeat("-", 30), strings.Repeat("-", 6),
		strings.Repeat("-", 4), strings.Repeat("-", 10),
		strings.Repeat("-", 30))

	// visit every file and directory in specified path
	err = filepath.Walk(os.Args[1],
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			var name string
			var errmsg string

			// print table row
			defer func() {
				if len(name) > 30 {
					name = name[:30]
				}
				fmt.Printf("%30s | %6s | %4s | %10v | %v\n", name,
					vcs.Mem.Cart.ID(),
					tv.GetFrameInfo().Spec.ID,
					mix.active,
					errmsg)
			}()

			// load and attach cartridge
			cartload, err := cartridgeloader.NewLoader(path, "AUTO")
			if err != nil {
				return err
			}
			if err := vcs.AttachCartridge(cartload, true); err != nil {
				name = filepath.Base(path)
				errmsg = err.Error()
				return nil
			}

			// reset VCS after previous iteration
			if err = vcs.Reset(); err != nil {
				return err
			}

			// run for 60 frames
			err = vcs.Run(func() (govern.State, error) {
				fr := tv.GetFrameInfo().FrameNum

				if fr > 60 {
					return govern.Ending, timedOut
				}

				return govern.Running, nil
			})

			// collect any errors
			if err != nil && !errors.Is(err, timedOut) {
				errmsg = err.Error()
			}

			// get short version of cartridge name
			name = cartload.ShortName()

			return nil
		})

	if err != nil {
		log.Fatal(err)
	}
}

type myAudioMixer struct {
	active bool
}

func (mix *myAudioMixer) SetAudio(sig []signal.SignalAttributes) error {
	for _, s := range sig {
		if s&signal.AudioUpdate != signal.AudioUpdate {
			continue
		}
		mix.active = uint8((s&signal.AudioChannel0)>>signal.AudioChannel0Shift) > 0 ||
			uint8((s&signal.AudioChannel1)>>signal.AudioChannel1Shift) > 0
	}
	return nil
}

func (mix *myAudioMixer) EndMixing() error {
	mix.active = false
	return nil
}

func (mix *myAudioMixer) Reset() {
}
