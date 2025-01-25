package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jetsetilly/gopher2600-utils/audit/auditors"
	"github.com/jetsetilly/gopher2600/archivefs"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

func clearLine() {
	fmt.Print("\r\033[2K")
}

func main() {
	// we don't want date/time in log entries
	log.SetFlags(0)

	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <path to ROMs>\n", os.Args[0])
	}
	pth := filepath.Clean(os.Args[1])

	// check path to roms argument
	f, err := os.Open(pth)
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

	// new VCS
	vcs, err := hardware.NewVCS(environment.MainEmulation, tv, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	// keep track of which roms have been audited. prevents reporting on
	// duplicate ROM files. key values are MD5 sums of cartridge data
	isAudited := make(map[string][]string)

	// message indicating the audit has succeeded
	const OkayMsg = "OK"

	auditResult := func(loader cartridgeloader.Loader, msg string) {
		// cropped filename
		fn := filepath.Clean(loader.Filename)
		fn, _ = strings.CutPrefix(fn, pth)
		fn, _ = strings.CutPrefix(fn, string(os.PathSeparator))

		// print message
		clearLine()
		fmt.Print(fn)
		if msg != OkayMsg {
			fmt.Printf("\n*** %s\n", msg)
		}

		// note that the ROM has been audited
		isAudited[loader.HashMD5] = append(isAudited[loader.HashMD5], loader.Name)
	}

	// auditing process
	auditf := func(loader cartridgeloader.Loader, audit auditors.Audit) error {
		err := vcs.AttachCartridge(loader, true)
		if err != nil {
			return err
		}

		if _, ok := isAudited[loader.HashMD5]; !ok {
			vcs.Mem.Cart.Reset()

			audit.Initialise(vcs)

			err := vcs.Run(func() (govern.State, error) {
				if err := audit.Check(); err != nil {
					return govern.Ending, err
				}
				return govern.Running, nil
			})

			if errors.Is(err, auditors.CheckEnded) {
				err = audit.Finalise()
				if errors.Is(err, auditors.FinalisedOk) {
					auditResult(loader, OkayMsg)
				} else {
					auditResult(loader, err.Error())
				}
			} else {
				auditResult(loader, err.Error())
			}

		}

		return nil
	}

	var afs archivefs.Path
	defer afs.Close()

	var walkf func(pth string) error
	walkf = func(pth string) error {
		err := afs.Set(pth, false)
		if err != nil {
			return err
		}
		if !afs.IsDir() {
			r, n, err := afs.Open()
			if err != nil {
				return err
			}

			data := make([]byte, n)
			_, err = r.Read(data)
			if err != nil {
				return err
			}

			loader, err := cartridgeloader.NewLoaderFromData(afs.String(), data, "AUTO", "AUTO", nil)
			if err != nil {
				return err
			}

			audit := auditors.HighHue{}
			_ = auditf(loader, &audit)
			return nil
		}

		lst, err := afs.List()
		if err != nil {
			return err
		}

		for _, l := range lst {
			err = walkf(filepath.Join(pth, l.Name))
			if err != nil {
				return err
			}
		}

		return nil
	}
	err = walkf(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}
}
