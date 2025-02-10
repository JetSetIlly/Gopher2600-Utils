package main

import (
	"encoding/json"

	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio/mix"
)

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

type soundEffects struct {
	gameName   string
	sampleRate int
	data       [][]byte
}

func parseJson(jsonFile []byte) (soundEffects, error) {
	var jsn gameJson

	err := json.Unmarshal(jsonFile, &jsn)
	if err != nil {
		return soundEffects{}, err
	}

	sfx := soundEffects{
		gameName:   jsn.Name,
		sampleRate: int(specification.SpecNTSC.HorizontalScanRate * audio.SamplesPerScanline),
	}

	var tia *audio.Audio
	tia = audio.NewAudio(nil)

	for _, sound := range jsn.SoundEffects {
		var data []byte

		for _, tone := range sound.Tones {
			tia.ReadMemRegisters(chipbus.ChangedRegister{
				Register: cpubus.AUDC0,
				Value:    uint8(tone.Control),
			})
			tia.ReadMemRegisters(chipbus.ChangedRegister{
				Register: cpubus.AUDF0,
				Value:    uint8(tone.Frequency),
			})
			tia.ReadMemRegisters(chipbus.ChangedRegister{
				Register: cpubus.AUDV0,
				Value:    uint8(tone.Volume),
			})

			// the assumed sound kernel is one update of the audio registers per frame
			for clk := range specification.SpecNTSC.ScanlinesTotal * specification.ClksScanline {
				if clk%3 == 0 && tia.Step() {
					m := mix.Mono(tia.Vol0, tia.Vol1)

					// sample is 16bits and added once for each stereo channel
					data = append(data, uint8(m))
					data = append(data, uint8(m>>8))
					data = append(data, uint8(m))
					data = append(data, uint8(m>>8))
				}
			}

		}

		sfx.data = append(sfx.data, data)
	}

	return sfx, nil
}
