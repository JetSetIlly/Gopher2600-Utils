package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jetsetilly/gopher2600-utils/audit/auditors"
	"github.com/jetsetilly/gopher2600/archivefs"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

type audit struct {
	// command line options
	recurse    bool
	concurrent bool

	// keep track of which roms have been audited. prevents reporting on
	// duplicate ROM files. key values are MD5 sums of cartridge data
	completed map[string][]string
}

func (aud *audit) run(pth string) error {
	// check path to roms argument
	f, err := os.Open(pth)
	defer f.Close()
	if err != nil {
		return err
	}

	var afs archivefs.Path
	defer afs.Close()

	auditResult := func(loader cartridgeloader.Loader, msg string) {
		// cropped filename
		fn := filepath.Clean(loader.Filename)
		fn, _ = strings.CutPrefix(fn, pth)
		fn, _ = strings.CutPrefix(fn, string(os.PathSeparator))

		const filenameColumnWidth = 48

		if len(fn) > filenameColumnWidth {
			fn = fn[len(fn)-filenameColumnWidth:]
		}
		fn = fmt.Sprintf("%s%s", fn, strings.Repeat(" ", filenameColumnWidth-len(fn)))

		// print message
		fmt.Printf("%s\t%s\n", fn, msg)

		// note that the ROM has been audited
		aud.completed[loader.HashMD5] = append(aud.completed[loader.HashMD5], loader.Name)
	}

	// auditing process
	auditf := func(loader cartridgeloader.Loader, audit auditors.Audit) error {
		// new television with auto-selecting tv protocl
		tv, err := television.NewTelevision("AUTO")
		if err != nil {
			return err
		}
		defer tv.End()
		tv.SetFPSCap(false)

		// new VCS
		vcs, err := hardware.NewVCS(environment.MainEmulation, tv, nil, nil)
		if err != nil {
			return err
		}

		err = vcs.AttachCartridge(loader, true)
		if err != nil {
			return err
		}

		if _, ok := aud.completed[loader.HashMD5]; !ok {
			vcs.Mem.Cart.Reset()

			audit.Initialise(vcs)

			err := vcs.Run(func() (govern.State, error) {
				if err := audit.Check(); err != nil {
					return govern.Ending, err
				}
				return govern.Running, nil
			})

			if errors.Is(err, auditors.CheckEnded) {
				var msg strings.Builder
				err = audit.Finalise(&msg)
				if errors.Is(err, auditors.FinalisedOk) {
					var s string
					if msg.Len() == 0 {
						s = "okay"
					} else {
						s = msg.String()
					}
					auditResult(loader, s)
				} else {
					auditResult(loader, err.Error())
				}
			} else {
				auditResult(loader, err.Error())
			}

		}

		return nil
	}

	var slots chan bool
	if aud.concurrent {
		slots = make(chan bool, runtime.NumCPU())
	} else {
		slots = make(chan bool, 1)
	}

	var walkf func(pth string, depth int) error
	walkf = func(pth string, depth int) error {
		// prevent recursion unless it's been activated
		if !aud.recurse && depth > 1 {
			return nil
		}

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

			slots <- true
			audit := auditors.COLUxxCount{}
			go func() {
				auditf(loader, &audit)
				<-slots
			}()
			return nil
		}

		lst, err := afs.List()
		if err != nil {
			return err
		}

		for _, l := range lst {
			err = walkf(filepath.Join(pth, l.Name), depth+1)
			if err != nil {
				return err
			}
		}

		return nil
	}

	return walkf(pth, 0)
}

func main() {
	// we don't want date/time in log entries
	log.SetFlags(0)

	aud := &audit{
		completed: make(map[string][]string),
	}

	// use flag set to provide the --help flag for top level command line.
	// that's all we want it to do
	flgs := flag.NewFlagSet("Gopher2600-Audit", flag.ExitOnError)

	// command line options
	flgs.BoolVar(&aud.recurse, "r", false, "recurse into directories")
	flgs.BoolVar(&aud.concurrent, "c", false, fmt.Sprintf("run audits concurrently (max: %d)", runtime.NumCPU()))

	// parse command line
	args := os.Args[1:]
	err := flgs.Parse(args)
	if err != nil {
		log.Fatal(err)
	}

	// treat all remainnig arguments as paths
	for _, pth := range flgs.Args() {
		pth = filepath.Clean(pth)
		err := aud.run(pth)
		if err != nil {
			log.Fatal(err)
		}
	}
}
