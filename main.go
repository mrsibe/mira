// Command mira is a ChatGPT-style desktop client powered by MyGo and GOAT.
package main

import (
	"log"
	"path/filepath"
	"runtime"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/mrsibe/mira/internal/chat"
	"github.com/mrsibe/mira/internal/config"
	"github.com/mrsibe/mira/internal/keychain"
	"github.com/mrsibe/mira/internal/memory"
	"github.com/mrsibe/mira/internal/service"
	"github.com/mrsibe/mira/internal/store"
	miratheme "github.com/mrsibe/mira/internal/theme"
	"github.com/mrsibe/mira/internal/updates"
)

var mainWindow atomic.Pointer[mygo.Window]

func main() {
	app := mygo.App
	app.SetName("Mira")

	dataDir, err := app.Path(mygo.PathUserData)
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := config.NewManager(dataDir, keychain.Store{})
	if err != nil {
		log.Fatal(err)
	}
	conversations, err := store.Open(filepath.Join(dataDir, "conversations"))
	if err != nil {
		log.Fatal(err)
	}
	memories, err := memory.Open(filepath.Join(dataDir, "memory"), conversations, chat.Complete)
	if err != nil {
		log.Fatal(err)
	}
	engine := chat.NewEngine()
	engine.Memory = memories
	mira := service.New(cfg, conversations, engine, memories)
	mygo.BindAs("Mira", mira)
	mygo.BindAs("MiraTheme", miratheme.New(applyTheme))
	mygo.BindAs("MiraUpdate", updates.New())
	mygo.Use(updater.New(updater.Options{Language: "zh-Hans"}))
	memories.MaybeStart(cfg.Settings())

	app.WhenReady(func() {
		app.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
		}))
		openWindow()
	})
	app.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			openWindow()
		}
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func openWindow() {
	options := mygo.WindowOptions{
		Title:           "Mira",
		Width:           1280,
		Height:          820,
		MinWidth:        800,
		MinHeight:       560,
		BackgroundColor: "light-dark(#ffffff, #212121)",
		TitleBarStyle:   mygo.TitleBarHidden,
		TitleBarHeight:  48,
		StateKey:        "main",
		URL:             "/",
	}
	if runtime.GOOS == "darwin" {
		options.TitleBarStyle = mygo.TitleBarHiddenInset
	}
	window := mygo.NewWindow(options)
	mainWindow.Store(window)
}

func applyTheme(mode string) error {
	source := mygo.ThemeSystem
	switch mode {
	case "light":
		source = mygo.ThemeLight
	case "dark":
		source = mygo.ThemeDark
	}
	mygo.Theme.SetSource(source)
	if window := mainWindow.Load(); window != nil {
		window.SetWindowButtonsTheme(source)
	}
	return nil
}
