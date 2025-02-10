package main

import (
	_ "embed"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed "aliens2600-sounds.json"
var aliensExample []byte

type emulator struct {
	audio *audio.Context
	sfx   soundEffects
}

func newEmulator() (*emulator, error) {
	var em emulator

	var err error
	em.sfx, err = parseJson(aliensExample)
	if err != nil {
		return nil, err
	}

	em.audio = audio.NewContext(em.sfx.sampleRate)

	return &em, nil
}

func (em *emulator) Update() error {
	var key int

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		key = 0
	} else if inpututil.IsKeyJustPressed(ebiten.Key2) {
		key = 1
	} else if inpututil.IsKeyJustPressed(ebiten.Key3) {
		key = 2
	} else if inpututil.IsKeyJustPressed(ebiten.Key4) {
		key = 3
	} else if inpututil.IsKeyJustPressed(ebiten.Key5) {
		key = 4
	} else if inpututil.IsKeyJustPressed(ebiten.Key6) {
		key = 5
	} else if inpututil.IsKeyJustPressed(ebiten.Key7) {
		key = 6
	} else if inpututil.IsKeyJustPressed(ebiten.Key8) {
		key = 7
	} else if inpututil.IsKeyJustPressed(ebiten.Key9) {
		key = 8
	} else {
		return nil
	}

	if key >= len(em.sfx.data) {
		return nil
	}

	if em.audio.IsReady() {
		ply := em.audio.NewPlayerFromBytes(em.sfx.data[key])
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
