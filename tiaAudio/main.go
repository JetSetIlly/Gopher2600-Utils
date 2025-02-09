package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	tia "github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio/mix"
)

//go:embed "aliens2600-sounds.json"
var example []byte

type gameJson struct {
	Name         string             `json:"gameName"`
	SoundEffects []soundEffectsJson `json:"soundEffects"`
}

type soundEffectsJson struct {
	Name  string      `json:"name"`
	Tones []tonesJson `json:"tones"`
}

type tonesJson struct {
	Control   int `json:"channel"`
	Volume    int `json:"volume"`
	Frequency int `json:"frequency"`
}

type emulator struct {
	tia   *tia.Audio
	audio *audio.Context
	sfx   gameJson
}

func newEmulator() (*emulator, error) {
	em := &emulator{
		tia:   tia.NewAudio(nil),
		audio: audio.NewContext(tia.OldSampleFreq),
	}

	err := json.Unmarshal(example, &em.sfx)
	if err != nil {
		return nil, err
	}

	return em, nil
}

func (em *emulator) Update() error {
	var play bool
	var sfx int
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		play = true
		sfx = 0
	} else if inpututil.IsKeyJustPressed(ebiten.Key2) {
		play = true
		sfx = 1
	} else if inpututil.IsKeyJustPressed(ebiten.Key3) {
		play = true
		sfx = 2
	} else if inpututil.IsKeyJustPressed(ebiten.Key4) {
		play = true
		sfx = 3
	} else if inpututil.IsKeyJustPressed(ebiten.Key5) {
		play = true
		sfx = 4
	} else if inpututil.IsKeyJustPressed(ebiten.Key6) {
		play = true
		sfx = 5
	} else if inpututil.IsKeyJustPressed(ebiten.Key7) {
		play = true
		sfx = 6
	} else if inpututil.IsKeyJustPressed(ebiten.Key8) {
		play = true
		sfx = 7
	} else if inpututil.IsKeyJustPressed(ebiten.Key9) {
		play = true
		sfx = 8
	}

	if play && sfx < len(em.sfx.SoundEffects) {
		var data []byte

		for _, tone := range em.sfx.SoundEffects[sfx].Tones {
			em.tia.ReadMemRegisters(chipbus.ChangedRegister{
				Register: cpubus.AUDC0,
				Value:    uint8(tone.Control),
			})
			em.tia.ReadMemRegisters(chipbus.ChangedRegister{
				Register: cpubus.AUDF0,
				Value:    uint8(tone.Frequency),
			})
			em.tia.ReadMemRegisters(chipbus.ChangedRegister{
				Register: cpubus.AUDV0,
				Value:    uint8(tone.Volume),
			})

			for i := range specification.SpecNTSC.ScanlinesTotal * specification.ClksScanline {
				if i%3 == 0 && em.tia.Step() {
					m := mix.Mono(em.tia.Vol0, em.tia.Vol1)
					data = append(data, uint8(m))
					data = append(data, uint8(m>>8))
					data = append(data, uint8(m))
					data = append(data, uint8(m>>8))
				}
			}
		}
		if !em.audio.IsReady() {
			return fmt.Errorf("audio context not ready")
		}

		ply := em.audio.NewPlayerFromBytes(data)
		ply.Play()
	}
	return nil
}

func (em *emulator) Draw(screen *ebiten.Image) {
}

func (em *emulator) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 10, 10
}

func main() {
	ebiten.SetWindowTitle("TIA Audio")
	ebiten.SetWindowDecorated(false)

	em, err := newEmulator()
	if err != nil {
		log.Fatal(err)
	}

	err = ebiten.RunGame(em)
	if err != nil {
		log.Fatal(err)
	}
}
