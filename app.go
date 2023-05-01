package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx      context.Context
	filename string
	url      string
	emit     chan struct {
		Type string
		Data interface{}
	}
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.emit = make(
		chan struct {
			Type string
			Data interface{}
		},
		10,
	)
	go func(a *App) {
		for {
			e := <-a.emit
			runtime.EventsEmit(a.ctx, e.Type, e.Data)
			time.Sleep(10 * time.Millisecond)
		}
	}(a)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) event(t string, d interface{}) {
	a.emit <- struct {
		Type string
		Data interface{}
	}{
		t,
		d,
	}
}

func (a *App) InitDownload(requestedURL string) error {
	a.event("progress", "Parsing URL")
	parsedURL, err := url.Parse(requestedURL)
	if err != nil {
		a.event("done", true)
		return err
	}
	_, err = url.ParseRequestURI(parsedURL.RequestURI())
	if err != nil {
		a.event("done", true)
		return err
	}

	a.event("progress", "Finding Default File Name")
	defaultFileName := filepath.Base(parsedURL.Path)
	switch defaultFileName {
	case ".":
		defaultFileName = ""
	case "/":
		defaultFileName = ""
	}
	a.url = requestedURL

	a.event("progress", "Asking where to save download file to")
	filename, err := runtime.SaveFileDialog(
		a.ctx,
		runtime.SaveDialogOptions{
			Title:                "Save As",
			DefaultFilename:      defaultFileName,
			CanCreateDirectories: true,
		},
	)
	if err != nil {
		a.event("done", true)
		return err
	}

	a.filename = filename
	return a.StartDownload()
}

func (a *App) StartDownload() error {
	defer a.event("done", true)
	bytes, canResume, err := a.head()
	if err != nil {
		return err
	}
	fp, err := os.OpenFile(a.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if !canResume {
		selected, err := runtime.MessageDialog(
			a.ctx,
			runtime.MessageDialogOptions{
				Title:   "Unable to resume",
				Message: "This URL does not support resuming files. Would you like to start the download over?",
				Buttons: []string{
					"Delete partial file and start over",
					"Never Mind",
				},
				DefaultButton: "Never Mind",
				CancelButton:  "Never Mind",
			},
		)
		if err != nil {
			return err
		}
		if "Never Mind" == selected {
			return fmt.Errorf("Download Aborted: not overwriting file")
		}
		fp.Truncate(0)
	}
	stat, _ := fp.Stat()
	read := stat.Size()
	p := new(progressBar)
	p.total = int64(bytes)
	p.current = read
	p.emit = a.event

	for {
		req, err := http.NewRequest(http.MethodGet, a.url, nil)
		if err != nil {
			return fmt.Errorf("Error preparing request to %s: %s", a.url, err.Error())
		}
		if read > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", read))
		}
		rsp, err := http.DefaultClient.Do(req)
		if rsp.StatusCode == 200 || rsp.StatusCode == 206 {
			if err != nil {
				return fmt.Errorf("Error requesting %s at byte offset %d: %s", a.url, read, err.Error())
			}
			got, _ := io.Copy(fp, io.TeeReader(rsp.Body, p))
			rsp.Body.Close()
			read += int64(got)
		}
		if read >= int64(bytes) {
			break
		}
		a.event("progress", "The download was interrupted. Retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	a.event("progress", "Download Complete")
	return nil
}

func (a *App) head() (int, bool, error) {
	rsp, err := http.Head(a.url)
	if err != nil {
		return 0, false, fmt.Errorf("Error making HEAD request to %s: %s", a.url, err.Error())
	}
	defer rsp.Body.Close()
	intLen, err := strconv.Atoi(rsp.Header.Get("Content-Length"))
	if err != nil {
		return 0, false, fmt.Errorf("Error reading or parsing Content-Length of '%s': %s", rsp.Header.Get("Content-Length"), err.Error())
	}
	if "bytes" != rsp.Header.Get("Accept-Ranges") {
		return intLen, false, fmt.Errorf("Missing valid Accept-Ranges header")
	}
	return intLen, true, nil
}
