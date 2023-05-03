package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/adrg/xdg"
)

const STATE_FILE_PATH = "github.com/apoklayptik/downloader/state.json"

type State struct {
	sync.Mutex
	filePath  string
	Downloads []*Downloader
}

func (s *State) Delete(entry *Downloader) {
	s.Lock()
	defer s.Unlock()
	newDownloads := []*Downloader{}
	for _, v := range s.Downloads {
		if v == entry {
			entry.cancel()
			entry.delete()
			continue
		}
		newDownloads = append(newDownloads, v)
	}
	s.Downloads = newDownloads
	s.Save()
}

func (s *State) Add(entry *Downloader) error {
	s.Lock()
	defer s.Unlock()
	for _, v := range s.Downloads {
		if v.URL == entry.URL {
			return fmt.Errorf("This URL has already been set to download: %s", entry.URL)
		}
	}
	s.Downloads = append(s.Downloads, entry)
	return s.Save()
}

func (s *State) Save() error {
	stateFilePath, err := xdg.DataFile(STATE_FILE_PATH)
	if err != nil {
		return err
	}
	buf, err := json.Marshal(s)
	dir, _ := filepath.Split(stateFilePath)
	if len(dir) == 0 {
		dir = "."
	}
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0666)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	return os.WriteFile(stateFilePath, buf, 0666)
}

func LoadState() (*State, error) {
	stateFilePath, err := xdg.DataFile(STATE_FILE_PATH)
	rval := &State{
		filePath:  stateFilePath,
		Downloads: []*Downloader{},
	}
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(stateFilePath)
	if os.IsNotExist(err) {
		return rval, rval.Save()
	}
	dir, fileName := filepath.Split(stateFilePath)
	if len(dir) == 0 {
		dir = "."
	}
	buf, err := fs.ReadFile(os.DirFS(dir), fileName)
	if err != nil {
		return rval, err
	}

	if len(buf) == 0 {
		return rval, rval.Save()
	}

	err = json.Unmarshal(buf, &rval)
	return rval, err
}
