package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s <path to ROMs>\n", os.Args[0])
		os.Exit(10)
	}

	// new television with auto-selecting tv protocl
	tv, err := television.NewTelevision("AUTO")
	if err != nil {
		fmt.Println(err)
		os.Exit(10)
	}
	defer tv.End()

	mix := &myAudioMixer{}
	tv.AddAudioMixer(mix)

	// new VCS
	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		fmt.Println(err)
		os.Exit(10)
	}

	// table header
	fmt.Printf("%s-+-%s-+-%s-+-%s-+-%v\n",
		strings.Repeat("-", 30), strings.Repeat("-", 6),
		strings.Repeat("-", 4), strings.Repeat("-", 5),
		strings.Repeat("-", 30))
	fmt.Printf("%30s | %6s | %4s | %5s | %v\n", "Name", "Format", "TV", "Audio", "Errors")
	fmt.Printf("%s-+-%s-+-%s-+-%s-+-%v\n",
		strings.Repeat("-", 30), strings.Repeat("-", 6),
		strings.Repeat("-", 4), strings.Repeat("-", 5),
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

			// load and attach cartridge
			cartload := cartridgeloader.Loader{Filename: path}
			if err := vcs.AttachCartridge(cartload); err != nil {
				if curated.Is(err, "cartridge: %v") {
					return nil
				}
				return err
			}

			// reset VCS after previous iteration
			if err = vcs.Reset(); err != nil {
				return err
			}

			// run for 60 frames
			err = vcs.Run(func() error {
				fr := tv.GetState(signal.ReqFramenum)

				if fr > 60 {
					return curated.Errorf("timed out")
				}

				return nil
			})

			// collect any errors
			errmsg := ""
			if err != nil && !curated.Is(err, "timed out") {
				errmsg = err.Error()
			}

			// get short version of cartridge name
			name := cartload.ShortName()
			if len(name) > 30 {
				name = name[:30]
			}

			// print table row
			fmt.Printf("%30s | %6s | %4s | %5v | %v\n", name,
				vcs.Mem.Cart.ID(),
				tv.GetSpec().ID,
				mix.active,
				errmsg)

			return nil
		})

	if err != nil {
		fmt.Println(err)
		os.Exit(10)
	}
}

type myAudioMixer struct {
	active bool
}

func (mix *myAudioMixer) SetAudio(data uint8) error {
	mix.active = data > 0
	return nil
}

func (mix *myAudioMixer) EndMixing() error {
	mix.active = false
	return nil
}

func (mix *myAudioMixer) Reset() {
}
