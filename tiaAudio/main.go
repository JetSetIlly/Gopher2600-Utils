package main

import (
	_ "embed"
	"log"
	"maps"
	"slices"
	"syscall/js"
)

//go:embed "aliens2600-sounds.json"
var aliensExample []byte

// results of parsing
var sfx soundEffects

func initialise(this js.Value, args []js.Value) interface{} {
	textarea := js.Global().Get("document").Call("getElementById", "json")
	if textarea.IsNull() || textarea.IsUndefined() {
		return nil
	}
	textarea.Set("value", string(aliensExample))

	updateSamples(this, args)
	return nil
}

func updateSamples(this js.Value, args []js.Value) interface{} {
	textarea := js.Global().Get("document").Call("getElementById", "json")
	if textarea.IsNull() || textarea.IsUndefined() {
		return nil
	}

	content := textarea.Get("value").String()

	var err error
	sfx, err = parseJson([]byte(content))
	if err != nil {
		log.Printf(err.Error())
		addError(err)
		return nil
	}

	log.Printf("name of game: %s", sfx.gameName)
	log.Printf("number of samples: %d", len(sfx.data))
	addButtons()

	return nil
}

func addError(err error) {
	contentDiv := js.Global().Get("document").Call("getElementById", "playback")
	if contentDiv.IsNull() || contentDiv.IsUndefined() {
		return
	}
	contentDiv.Set("innerHTML", err.Error())
}

func addButtons() {
	contentDiv := js.Global().Get("document").Call("getElementById", "playback")
	if contentDiv.IsNull() || contentDiv.IsUndefined() {
		return
	}
	contentDiv.Set("innerHTML", "")

	keys := slices.Collect(maps.Keys(sfx.data))
	slices.Sort(keys)
	for _, name := range keys {
		button := js.Global().Get("document").Call("createElement", "button")
		button.Set("innerHTML", name) // Set the button's text

		button.Call("addEventListener", "click", js.FuncOf(func(this js.Value, p []js.Value) interface{} {
			playSample(name)
			return nil
		}))

		contentDiv.Call("appendChild", button)
	}
}

func playSample(name string) {
	sample := sfx.data[name]
	log.Printf("playing %s at %.2fHz for %d frames", name, sfx.sampleRate, len(sample))

	// Get the Web Audio API context
	audioContext := js.Global().Get("AudioContext").New()

	audioBuffer := audioContext.Call("createBuffer", sfx.channels, len(sample)*sfx.channels,
		sfx.sampleRate*float32(sfx.channels*sfx.size))

	leftChannel := audioBuffer.Call("getChannelData", 0)
	rightChannel := audioBuffer.Call("getChannelData", 1)

	// gain of audio output is too high for us so we directly reduce the gain on the samples
	const gain = 0.01

	for i := range sample {
		idx := i * 2
		leftChannel.SetIndex(idx, (float64(sample[i])-128.0)/128.0*gain)
		rightChannel.SetIndex(idx+1, (float64(sample[i])-128.0)/128.0*gain)
	}

	source := audioContext.Call("createBufferSource")
	source.Set("buffer", audioBuffer)

	// connect directly to audio output
	source.Call("connect", audioContext.Get("destination"))
	source.Call("start", 0)
}

func main() {
	js.Global().Set("initialise", js.FuncOf(initialise))
	js.Global().Set("updateSamples", js.FuncOf(updateSamples))

	select {}
}
