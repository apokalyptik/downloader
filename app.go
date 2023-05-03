package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DownloaderBackend struct
type DownloaderBackend struct {
	ctx      context.Context
	filename string
	url      string
	state    *State
	ticking  bool
}

// NewApp creates a new App application struct
func NewApp() *DownloaderBackend {
	state, err := LoadState()
	if err != nil {
		panic(err)
	}
	return &DownloaderBackend{
		state: state,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (d *DownloaderBackend) startup(ctx context.Context) {
	d.ctx = ctx
}

func (d *DownloaderBackend) FrontEndReady() {
	event.ctx = d.ctx
	go event.handle()
	eventChan <- Event{Type: "updateDownloads", Data: d.state.Downloads}
	go d.tick()
}

func (d *DownloaderBackend) genericError(title, errorStr string) {
	runtime.MessageDialog(
		d.ctx,
		runtime.MessageDialogOptions{
			Title:   title,
			Message: errorStr,
			Buttons: []string{"OK"},
		},
	)
}

func (d *DownloaderBackend) Add(requestedURL string) error {
	dl, err := NewDownloader(requestedURL)
	if err != nil {
		d.genericError("We ran into an error", err.Error())
		return err
	}
	if err := d.state.Add(dl); err != nil {
		d.genericError("We ran into an error", err.Error())
		return err
	}
	eventChan <- Event{Type: "setUrl", Data: ""}
	eventChan <- Event{Type: "updateDownloads", Data: d.state.Downloads}
	return nil
}

func (d *DownloaderBackend) Pause(url string, state bool) {
	d.state.Lock()
	defer d.state.Unlock()
	for _, v := range d.state.Downloads {
		if v.URL == url {
			v.Paused = state
			if state {
				v.cancel()
			}
			break
		}
	}
}

func (d *DownloaderBackend) Delete(url string) {
	ays, err := runtime.MessageDialog(
		d.ctx,
		runtime.MessageDialogOptions{
			Title:         "Remove this download?",
			Message:       "Are you sure you would like to remove this download?",
			Buttons:       []string{"Delete", "Cancel"},
			DefaultButton: "Cancel",
			CancelButton:  "Cancel",
		},
	)
	if err != nil {
		return
	}
	if ays != "Delete" {
		return
	}
	var entry *Downloader
	d.state.Lock()
	for _, v := range d.state.Downloads {
		if v.URL == url {
			entry = v
			break
		}
	}
	d.state.Unlock()
	if entry != nil {
		d.state.Delete(entry)
		eventChan <- Event{Type: "updateDownloads", Data: d.state.Downloads}
	}
}

func (d *DownloaderBackend) Save(url string) {
	var entry *Downloader
	d.state.Lock()
	for _, v := range d.state.Downloads {
		if v.URL == url {
			entry = v
			break
		}
	}
	d.state.Unlock()
	if entry != nil {
		defaultFilename := entry.FilenameGuessFromURL
		if entry.FilenameGuessFromHEAD != "" {
			defaultFilename = entry.FilenameGuessFromHEAD
		}
		as, err := runtime.SaveFileDialog(
			d.ctx,
			runtime.SaveDialogOptions{
				Title:           "Save this file as",
				DefaultFilename: defaultFilename,
			},
		)
		if err != nil {
			d.genericError("Error getting filename to save as", err.Error())
			return
		}
		err = os.Rename(entry.TempFile, as)
		if err != nil {
			d.genericError("Error saving file", fmt.Sprintf("Error saving file %s\n\n%s", as, err.Error()))
			return
		}
		d.state.Delete(entry)
		eventChan <- Event{Type: "updateDownloads", Data: d.state.Downloads}
	}
}

func (d *DownloaderBackend) tick() {
	if d.ticking {
		return
	}
	t := time.Tick(200 * time.Millisecond)
	s := time.Tick(time.Second)
	for {
		select {
		case <-t:
			d.state.Lock()
			for _, v := range d.state.Downloads {
				if v.Paused {
					continue
				}
				if v.isAttempting {
					v.snap()
					continue
				}
				if v.Complete {
					continue
				}
				go v.attempt()
			}
			eventChan <- Event{Type: "updateDownloads", Data: d.state.Downloads}
			d.state.Unlock()
			break
		case <-s:
			d.state.Lock()
			d.state.Save()
			d.state.Unlock()
			break
		}
	}
}
