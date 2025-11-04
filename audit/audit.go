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
	"sync"

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
	auditor    string

	// keep track of which roms have been audited. prevents reporting on
	// duplicate ROM files. key values are MD5 sums of cartridge data
	completed map[string][]string
}

func (aud *audit) run(pth string) error {
	// check path to roms argument
	f, err := os.Open(pth)
	if err != nil {
		return err
	}

	// for presentation purposes we want to cut the path from the audited ROM
	// file. what we consider to the prefix depends on whether the pth parameter
	// is a directory or a file
	var prefix string

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		prefix = pth
	} else {
		prefix = filepath.Base(pth)
	}

	var afs archivefs.Path
	defer afs.Close()

	auditResult := func(loader cartridgeloader.Loader, msg string) {
		// cropped filename
		fn := filepath.Clean(loader.Filename)
		fn, _ = strings.CutPrefix(fn, prefix)
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

		err = vcs.AttachCartridge(loader)
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

	// counts active audit goroutines
	var wg sync.WaitGroup

	// number of emulation slots depending on concurrency argument
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

			// create new auditor instance. validity of auditor id should have
			// been checked already
			audit := auditors.Factory[aud.auditor]()

			wg.Add(1)
			slots <- true
			go func() {
				_ = auditf(loader, audit)
				<-slots
				wg.Done()
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

	err = walkf(pth, 0)
	if err != nil {
		return err
	}

	// wait until all audit goroutines are finished
	wg.Wait()

	return nil
}

func main() {
	// we don't want date/time in log entries
	log.SetFlags(0)

	aud := &audit{
		completed: make(map[string][]string),
	}

	// command line options
	flgs := flag.NewFlagSet("Gopher2600-Audit", flag.ContinueOnError)

	flgs.BoolVar(&aud.recurse, "r", false, "recurse into directories")
	flgs.BoolVar(&aud.concurrent, "c", false, fmt.Sprintf("run audits concurrently (max: %d)", runtime.NumCPU()))
	flgs.StringVar(&aud.auditor, "a", auditors.Factory[auditors.DefaultAuditor]().ID(), "which auditor to run")

	// parse command line
	args := os.Args[1:]
	err := flgs.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Print("\nAuditors: ")
			for key, aud := range auditors.Factory {
				if key != auditors.DefaultAuditor {
					fmt.Print(aud().ID(), " ")
				}
			}
			fmt.Println("")
			return
		}
		log.Fatal(err)
	}

	// check that selected auditor is valid
	n := auditors.NormaliseID(aud.auditor)
	if _, ok := auditors.Factory[n]; !ok {
		log.Fatalf("*** invalid auditor: %s", aud.auditor)
	}
	aud.auditor = n

	// treat all remaining arguments as paths
	for _, pth := range flgs.Args() {
		pth = filepath.Clean(pth)
		err := aud.run(pth)
		if err != nil {
			log.Fatal(err)
		}
	}
}
