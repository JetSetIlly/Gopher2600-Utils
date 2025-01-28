// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"

	_ "embed"
)

//go:embed "example.bin"
var example_bin []byte
var example_title = "RSBoxing"

// horizontal scaling of image
const pixelWidth = 2

type emulator struct {
	tv   *television.Television
	spec specification.Spec
	vcs  *hardware.VCS

	width  int // width of image (for presentation multiply by pixelWidth)
	height int
	top    int
	bottom int

	frameNum int

	imageCrit sync.Mutex
	image     *ebiten.Image
	pixels    []byte
}

func runEmulator() *emulator {
	em := &emulator{}

	var err error

	em.tv, err = television.NewTelevision("NTSC")
	if err != nil {
		println(err.Error())
		return nil
	}
	em.tv.SetFPSCap(false)

	em.vcs, err = hardware.NewVCS(environment.MainEmulation, em.tv, nil, nil)
	if err != nil {
		println(err.Error())
		return nil
	}

	em.tv.AddPixelRenderer(em)

	em.Resize(television.NewFrameInfo(specification.SpecNTSC))

	err = em.vcs.Reset()
	if err != nil {
		println(err.Error())
		return nil
	}

	loader, err := cartridgeloader.NewLoaderFromData(example_title, example_bin, "AUTO", "AUTO", nil)
	if err != nil {
		println(err.Error())
		return nil
	}

	err = em.vcs.AttachCartridge(loader, true)
	if err != nil {
		println(err.Error())
		return nil
	}

	return em
}

func (em *emulator) Update() error {
	return em.vcs.RunForFrameCount(1, nil)
}

func (em *emulator) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(pixelWidth, 1)

	em.imageCrit.Lock()
	defer em.imageCrit.Unlock()

	screen.DrawImage(em.image, op)
}

func (em *emulator) Layout(outsideWidth, outsideHeight int) (int, int) {
	return em.width * pixelWidth, em.height
}

// Resize implements television.PixelRenderer
func (em *emulator) Resize(frameInfo television.FrameInfo) error {
	ebiten.SetTPS(int(frameInfo.RefreshRate))

	em.spec = frameInfo.Spec
	em.top = frameInfo.VisibleTop
	em.bottom = frameInfo.VisibleBottom
	em.height = em.bottom - em.top

	// strictly, only the height will ever change on a specification change but
	// it's convenient to set the width too
	em.width = specification.ClksVisible

	em.image = ebiten.NewImage(em.width, em.height)
	if em.image == nil {
		return errors.New("unable to allocate backing image")
	}

	// set alpha channel on creation - it never changes
	em.pixels = make([]byte, em.width*em.height*4)
	for i := 3; i <= len(em.pixels); i += 4 {
		em.pixels[i] = 255
	}

	return nil
}

// NewFrame implements television.PixelRenderer
func (em *emulator) NewFrame(_ television.FrameInfo) error {
	em.frameNum++
	if em.frameNum%60 == 0 {
		fps, hz := em.tv.GetActualFPS()
		fmt.Printf("Ebiten: %.1ffps   Emulator: %.1ffps (%.1fHz)\n", ebiten.ActualFPS(), fps, hz)
	}

	return nil
}

// NewScanline implements television.PixelRenderer
func (em *emulator) NewScanline(scanline int) error {
	return nil
}

// SetPixel implements television.PixelRenderer
func (em *emulator) SetPixels(sig []signal.SignalAttributes, last int) error {
	em.imageCrit.Lock()
	defer em.imageCrit.Unlock()

	i := 0
	for _, s := range sig {
		sl := s.Index / specification.ClksScanline
		cl := s.Index % specification.ClksScanline
		x := cl - specification.ClksHBlank
		y := sl - em.top

		if x < 0 || y < 0 {
			continue
		}

		var rgb color.RGBA

		// handle VBLANK by setting pixels to black. we also manually handle
		// NoSignal in the same way
		if s.VBlank || s.Index == signal.NoSignal {
			rgb = em.spec.GetColor(signal.VideoBlack)
		} else {
			rgb = em.spec.GetColor(s.Color)
		}

		// skip alpha channel - it never changes
		s := em.pixels[i : i+3 : i+3]
		s[0] = rgb.R
		s[1] = rgb.G
		s[2] = rgb.B
		i += 4

		if i >= len(em.pixels) {
			break
		}
	}

	em.image.WritePixels(em.pixels)

	return nil
}

// Reset implements television.PixelRenderer
func (em *emulator) Reset() {
}

// EndRendering implements television.PixelRenderer
func (em *emulator) EndRendering() error {
	return nil
}

func profile(runFunc func()) {
	cpuf, err := os.Create("cpu.profile")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := cpuf.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	err = pprof.StartCPUProfile(cpuf)
	if err != nil {
		log.Fatal(err)
	}
	defer pprof.StopCPUProfile()

	memf, err := os.Create("mem.profile")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := memf.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	defer func() {
		runtime.GC()
		err = pprof.WriteHeapProfile(memf)
		if err != nil {
			log.Print(err.Error())
		}
	}()

	runFunc()
}

func run() {
	ebiten.SetWindowTitle("Gopher2600")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(runEmulator()); err != nil {
		log.Fatal(err)
	}
}

func main() {
	// profile(run)
	run()
}
