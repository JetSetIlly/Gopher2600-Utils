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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// +build js
// +build wasm

package main

import (
	"image"
	"syscall/js"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const pixelWidth = 2
const horizScale = 2
const vertScale = 2

// Canvas implements television.PixelRenderer
type Canvas struct {
	// the worker in which our WASM application is running
	worker js.Value

	tv   *television.Television
	spec specification.Spec

	width  int
	height int
	top    int
	bottom int

	frameNum int

	image *image.RGBA
	done  bool
}

// NewCanvas is the preferred method of initialisation for the Canvas type
func NewCanvas(worker js.Value) (*Canvas, error) {
	var err error

	scr := &Canvas{worker: worker}

	scr.tv, err = television.NewTelevision("NTSC")
	if err != nil {
		return nil, err
	}
	defer scr.tv.End()

	scr.tv.AddPixelRenderer(scr)

	return scr, nil
}

// Resize implements television.PixelRenderer
func (scr *Canvas) Resize(spec specification.Spec, topScanline, bottomScanline int) error {
	if scr.done {
		return nil
	}
	scr.done = true
	scr.spec = spec
	scr.top = topScanline
	scr.bottom = bottomScanline
	scr.height = (bottomScanline - topScanline) * vertScale

	// strictly, only the height will ever change on a specification change but
	// it's convenient to set the width too
	scr.width = specification.ClksVisible * pixelWidth * horizScale

	scr.image = image.NewRGBA(image.Rect(0, 0, scr.width, scr.height))

	// resize HTML canvas
	scr.worker.Call("updateCanvasSize", scr.width, scr.height)

	return nil
}

// NewFrame implements television.PixelRenderer
func (scr *Canvas) NewFrame(isStable bool) error {
	scr.frameNum++
	scr.worker.Call("updateDebug", "frameNum", scr.frameNum)

	pixels := js.Global().Get("Uint8Array").New(len(scr.image.Pix))
	js.CopyBytesToJS(pixels, scr.image.Pix)
	scr.worker.Call("updateCanvas", pixels)

	// give way to messageHandler - there must be a more elegant way of doing this
	time.Sleep(25 * time.Millisecond)

	return nil
}

// NewScanline implements television.PixelRenderer
func (scr *Canvas) NewScanline(scanline int) error {
	// scr.worker.Call("updateDebug", "scanline", scanline)
	return nil
}

// UpdatingPixels implements television.PixelRenderer
func (scr *Canvas) UpdatingPixels(_ bool) {
}

// SetPixel implements television.PixelRenderer
func (scr *Canvas) SetPixel(sig signal.SignalAttributes, current bool) error {
	// we could return immediately but if vblank is on inside the visible
	// area we need to the set pixel to black, in case the vblank was off
	// in the previous frame (for efficiency, we're not clearing the pixel
	// array at the end of the frame)

	// adjust pixels so we're only dealing with the visible range
	x := sig.Clock - specification.ClksHBlank
	y := sig.Scanline - scr.top

	if x < 0 || y < 0 {
		return nil
	}

	rgb := scr.spec.GetColor(sig.Pixel)

	for h := 0; h < vertScale; h++ {
		for w := 0; w < horizScale*pixelWidth; w++ {
			scr.image.SetRGBA(
				(x*horizScale*pixelWidth)+w,
				(y*vertScale)+h,
				rgb)
		}
	}

	return nil
}

// Reset implements television.PixelRenderer
func (scr *Canvas) Reset() {
}

// EndRendering implements television.PixelRenderer
func (scr *Canvas) EndRendering() error {
	return nil
}
