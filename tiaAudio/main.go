package main

import (
	_ "embed"
	"log"
	"maps"
	"slices"
	"syscall/js"
	"time"
)

//go:embed "aliens2600-sounds.json"
var aliensExample []byte

// results of parsing
var sfx soundEffects

func initialiseWithExampleJSON(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		log.Printf("tiaAudio: too many arguments to initialiseWithExampleJSON()")
		return nil
	}

	textarea := js.Global().Get("document").Call("getElementById", "json")
	if textarea.IsNull() || textarea.IsUndefined() {
		return nil
	}
	textarea.Set("value", string(aliensExample))

	content := js.ValueOf(string(aliensExample))
	updateSamples(this, []js.Value{content})
	return nil
}

func updateSamples(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		log.Printf("tiaAudio: updateSamples() called with no json data")
		return nil
	} else if len(args) > 1 {
		log.Printf("tiaAudio: too many arguments to updateSamples()")
		return nil
	}
	content := args[0].String()

	addMessage("emulating...")
	startTime := time.Now()

	var err error
	sfx, err = parseJson([]byte(content))
	if err != nil {
		log.Printf("tiaAudio: %s", err.Error())
		addMessage(err.Error())
		return nil
	}

	log.Printf("tiaAudio: emulation time %s", time.Since(startTime))
	log.Printf("tiaAudio: name of game: %s", sfx.gameName)
	log.Printf("tiaAudio: number of samples: %d", len(sfx.data))

	addButtons()

	return nil
}

func addMessage(msg string) {
	contentDiv := js.Global().Get("document").Call("getElementById", "userFeedback")
	if contentDiv.IsNull() || contentDiv.IsUndefined() {
		return
	}
	contentDiv.Set("innerHTML", msg)
}

func addButtons() {
	contentDiv := js.Global().Get("document").Call("getElementById", "userFeedback")
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
			n := js.ValueOf(string(name))
			playSample(js.Undefined(), []js.Value{n})
			return nil
		}))

		contentDiv.Call("appendChild", button)
	}
}

func playSample(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		log.Printf("tiaAudio: playSample() called with no sample name")
		return nil
	} else if len(args) > 1 {
		log.Printf("tiaAudio: too many arguments to playSample()")
		return nil
	}
	name := args[0].String()

	sample, ok := sfx.data[name]
	if !ok {
		log.Printf("tiaAudio: unknown sample '%s'", name)
		return nil
	}

	log.Printf("tiaAudio: playing %s at %.2fHz for %d frames", name, sfx.sampleRate, len(sample))

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

	return nil
}

func main() {
	js.Global().Set("initialiseWithExampleJSON", js.FuncOf(initialiseWithExampleJSON))
	js.Global().Set("updateSamples", js.FuncOf(updateSamples))
	js.Global().Set("playSample", js.FuncOf(playSample))

	select {}
}
