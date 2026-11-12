// Command mira is a ChatGPT-style desktop client powered by MyGo and GOAT.
package main

import (
	"log"
	"net/url"
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

const disableBrowserZoom = `
(() => {
	const stopZoomWheel = (event) => {
		if (!event.ctrlKey && !event.metaKey) return;
		event.preventDefault();
		event.stopPropagation();
	};
	const stopZoomKey = (event) => {
		if (!event.ctrlKey && !event.metaKey) return;
		if (event.key !== "+" && event.key !== "=" && event.key !== "-" && event.key !== "_" && event.key !== "0") return;
		event.preventDefault();
		event.stopPropagation();
	};
	window.addEventListener("wheel", stopZoomWheel, { passive: false, capture: true });
	window.addEventListener("keydown", stopZoomKey, { capture: true });
})();
`

func main() {
	app := mygo.App
	if mygo.IsDev() {
		app.SetName("Mira Dev")
	}
	if !app.RequestSingleInstanceLock() {
		return
	}
	app.OnSecondInstance(func([]string, string) { showMainWindow() })
	app.SetDMABUFRenderer(mygo.DMABUFRendererAuto)

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
			{Role: mygo.RoleEditMenu},
		}))
		showMainWindow()
	})
	app.OnWindowAllClosed(func() {
		if runtime.GOOS != "darwin" {
			app.Quit()
		}
	})
	app.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			showMainWindow()
		}
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func showMainWindow() {
	if focusOpenWindow() {
		return
	}
	mygo.App.WhenReady(func() {
		if focusOpenWindow() {
			return
		}
		openWindow()
	})
}

func focusOpenWindow() bool {
	window := mainWindow.Load()
	if window == nil {
		return false
	}
	if window.IsDestroyed() {
		mainWindow.CompareAndSwap(window, nil)
		return false
	}
	if window.IsMinimized() {
		window.Restore()
	}
	window.Show()
	window.Focus()
	mygo.App.Focus()
	return true
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
		Page: mygo.PageOptions{
			PreloadScript: disableBrowserZoom,
		},
	}
	if runtime.GOOS == "darwin" {
		options.TitleBarStyle = mygo.TitleBarHiddenInset
	}
	window := mygo.NewWindow(options)
	mainWindow.Store(window)
	window.OnClosed(func() {
		mainWindow.CompareAndSwap(window, nil)
	})
	window.Page().OnWillNavigate(func(event *mygo.NavigateEvent) {
		if !event.UserInitiated {
			return
		}
		target, err := url.Parse(event.URL)
		if err != nil || (target.Scheme != "http" && target.Scheme != "https") {
			return
		}
		event.PreventDefault()
		go func() {
			if err := mygo.Shell.OpenExternal(event.URL); err != nil {
				log.Printf("open %s: %v", event.URL, err)
			}
		}()
	})
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
