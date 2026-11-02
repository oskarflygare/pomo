//go:build darwin && cgo

package menubar

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#include "native_darwin.h"
*/
import "C"

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/Bahaaio/pomo/config"
	"github.com/Bahaaio/pomo/db"
)

func init() {
	// AppKit must run on the main thread
	runtime.LockOSThread()
}

var (
	controller  *Controller
	menuActions map[int]func() // only accessed on the main thread
)

// Run shows the menu bar timer and blocks until it quits.
func Run(cfg config.Config) error {
	var repo Repo
	if database, err := db.Connect(); err != nil {
		log.Printf("failed to initialize database: %v", err)
	} else {
		repo = db.NewSessionRepo(database)
	}

	var login LoginToggle
	if item, err := DefaultLoginItem(); err != nil {
		log.Printf("login item unavailable: %v", err)
	} else {
		login = item
	}

	controller = NewController(cfg, repo, nativeView{}, login, func() { C.pomoQuit() })

	// record the current session when interrupted from the terminal
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		controller.Quit()
	}()

	C.pomoRun()
	return nil
}

type nativeView struct{}

func (nativeView) SetStatus(s Status) {
	label := C.CString(s.Label)
	defer C.free(unsafe.Pointer(label))

	C.pomoSetStatus(label, C.double(s.Progress), C.bool(s.Paused))
}

//export goReady
func goReady() {
	controller.Tick()

	go func() {
		for range time.Tick(time.Second) {
			controller.Tick()
		}
	}()
}

//export goRightClick
func goRightClick() {
	controller.Toggle()
}

//export goLeftClick
func goLeftClick() {
	menuActions = map[int]func(){}
	nextID := 0

	var add func(parent int, items []MenuItem)
	add = func(parent int, items []MenuItem) {
		for _, item := range items {
			id := nextID
			nextID++

			if item.Action != nil {
				menuActions[id] = item.Action
			}

			title := C.CString(item.Title)
			detail := C.CString(item.Detail)
			C.pomoMenuAdd(C.int(parent), C.int(id), title, detail,
				C.bool(item.Enabled), C.bool(item.Checked), C.bool(item.Separator), C.bool(item.Action != nil))
			C.free(unsafe.Pointer(title))
			C.free(unsafe.Pointer(detail))

			if len(item.Submenu) > 0 {
				add(id, item.Submenu)
			}
		}
	}

	C.pomoMenuBegin()
	add(-1, controller.Menu())
	C.pomoMenuShow()
}

//export goMenuAction
func goMenuAction(id C.int) {
	if action, ok := menuActions[int(id)]; ok {
		action()
	}
}
