package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/adrg/xdg"
)

const DOWNLOAD_TEMP_PATH = "github.com/apoklayptik/downloader/tmp/"

type Downloader struct {
	sync.Mutex
	ctx       context.Context
	ctxCancel context.CancelFunc

	URL                   string
	TempFile              string
	FilenameGuessFromURL  string
	FilenameGuessFromHEAD string
	Bytes                 int64
	DownloadedBytes       int64
	Resumable             bool
	Paused                bool
	Complete              bool
	Pct                   float32
	BytesPerSecond        int
	AttemptHeaders        http.Header
	LastAttempt           time.Time
	AttemptCounter        int

	headReqError error

	isReady      bool
	isAttempting bool
	started      bool
	attemptError error
	attemptRsp   *http.Response

	// Progress Tracking
	nextSnap      time.Time
	snapTime      time.Time
	snapBytes     int64
	snapBpsBucket []int
}

func (d *Downloader) cancel() {
	if d.ctxCancel != nil {
		d.ctxCancel()
	}
}

func (d *Downloader) delete() {
	// Make sure we can write the temp file
	_, err := os.Stat(d.TempFile)
	if err == nil {
		os.Remove(d.TempFile)
	}

}

func (d *Downloader) attempt() error {
	if d.isAttempting {
		return nil
	}

	d.Lock()
	defer d.Unlock()

	now := time.Now()
	if now.Sub(d.LastAttempt) < (5 * time.Second) {
		return nil
	}
	d.LastAttempt = now

	d.AttemptCounter++

	d.ctx, d.ctxCancel = context.WithCancel(context.Background())

	d.isAttempting = true

	fp, err := os.OpenFile(d.TempFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		d.isAttempting = false
		return err
	}
	defer fp.Close()
	stat, _ := fp.Stat()
	d.DownloadedBytes = stat.Size()

	log.Println("beginning download ", d.URL, d.TempFile)
	req, err := http.NewRequestWithContext(d.ctx, http.MethodGet, d.URL, nil)
	if err != nil {
		return fmt.Errorf("Error preparing request to %s: %s", d.URL, err.Error())
	}
	if d.DownloadedBytes > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", d.DownloadedBytes))
	}

	d.attemptRsp, d.attemptError = http.DefaultClient.Do(req)
	if d.attemptRsp != nil && d.attemptRsp.Body != nil {
		defer d.attemptRsp.Body.Close()
	}

	if d.attemptError != nil {
		d.isAttempting = false
		return fmt.Errorf("Error requesting %s at byte offset %d: %s", d.URL, d.DownloadedBytes, d.attemptError.Error())
	}
	d.AttemptHeaders = d.attemptRsp.Header
	if d.attemptRsp.StatusCode == 200 || d.attemptRsp.StatusCode == 206 {
		if d.resumable(d.AttemptHeaders) {
			d.Resumable = true
		} else {
			d.Resumable = false
			fp.Truncate(0)
			d.DownloadedBytes = 0
		}

		if d.DownloadedBytes == 0 {
			d.Bytes = d.bytes(d.AttemptHeaders)
		} else {
			d.Bytes = d.DownloadedBytes + d.bytes(d.AttemptHeaders)
		}

		if d.FilenameGuessFromHEAD == "" {
			d.FilenameGuessFromHEAD = d.dispositionFliename(d.AttemptHeaders)
		}
		io.Copy(fp, io.TeeReader(d.attemptRsp.Body, d))
	}

	if d.DownloadedBytes >= d.Bytes {
		d.Complete = true
		d.Pct = 100

	}
	d.isAttempting = false

	return nil
}

func (d *Downloader) dispositionFliename(h http.Header) string {
	contentDisposition := h.Get("Content-Disposition")
	if contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			if guess, ok := params["filename"]; ok {
				return guess
			}
		}
	}
	return ""
}

func (d *Downloader) resumable(h http.Header) bool {
	if "bytes" == h.Get("Accept-Ranges") {
		return true
	} else {
		return true
	}
}

func (d *Downloader) bytes(h http.Header) int64 {
	b, _ := strconv.Atoi(h.Get("Content-Length"))
	return int64(b)
}

// Write implements the io.Writer interface so that we can use an io.TeeReader during the download
// on ourselves to record the number of bytes written so far.
func (d *Downloader) Write(b []byte) (int, error) {
	l := len(b)
	d.DownloadedBytes = d.DownloadedBytes + int64(l)
	if !d.started {
		d.started = true
		d.snapBytes = d.DownloadedBytes
		d.snapTime = time.Now()
		d.snapBpsBucket = []int{}
	}
	return l, nil
}

func (d *Downloader) snap() {
	if d.DownloadedBytes < 1 {
		return
	}
	now := time.Now()
	downloadedBytes := d.DownloadedBytes
	if d.Bytes > 0 {
		d.Pct = (float32(downloadedBytes) / float32(d.Bytes)) * 100
	} else {
		d.Pct = 0
	}

	snapBytes := d.snapBytes
	sec := now.Sub(d.snapTime).Seconds()
	newBytesPerSecond := int(float64(downloadedBytes-snapBytes) / sec)
	vals := 1
	val := newBytesPerSecond
	if len(d.snapBpsBucket) > 0 {
		for _, v := range d.snapBpsBucket {
			vals++
			val = val + v
		}
	}
	d.BytesPerSecond = int(val / vals)

	if now.Before(d.nextSnap) && len(d.snapBpsBucket) > 0 {
		return
	}
	d.nextSnap = now.Add(time.Second)
	d.snapBytes = downloadedBytes
	d.snapTime = now
	newSnapBpsBucket := append(d.snapBpsBucket, newBytesPerSecond)
	if len(newSnapBpsBucket) > 10 {
		newSnapBpsBucket = newSnapBpsBucket[1:]
	}
	d.snapBpsBucket = newSnapBpsBucket
}

func NewDownloader(URL string) (*Downloader, error) {
	dl := new(Downloader)
	dl.URL = URL

	// Validate the given URL
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	// Generate a unique temporary filename
	hasher := md5.New()
	fmt.Fprintf(hasher, URL)
	tempName := fmt.Sprintf("%0x", string(hasher.Sum(nil)))
	dl.TempFile, err = xdg.StateFile(fmt.Sprintf("%s%s.tmp", DOWNLOAD_TEMP_PATH, tempName))
	if err != nil {
		return nil, err
	}

	// Make sure we can write the temp dir
	dir, _ := filepath.Split(dl.TempFile)
	if len(dir) == 0 {
		dir = "."
	}
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0666)
		if err != nil {
			return nil, err
		}
	}

	// Make sure we can write the temp file
	_, err = os.Stat(dl.TempFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// the file doesn't exist, so attempt to create it...
		file, err := os.Create(dl.TempFile)
		if file != nil {
			file.Close()
		}
		if err != nil {
			return nil, err
		}
	} else {
		// the file exists, so attempt to open it in RW mode...
		file, err := os.OpenFile(dl.TempFile, os.O_RDWR, 0666)
		if file != nil {
			defer file.Close()
		}
		if err != nil {
			return nil, err
		}
	}

	// Attempt to guess the destination filename based on the provided URL
	dl.FilenameGuessFromURL = filepath.Base(parsedURL.Path)
	if "." == dl.FilenameGuessFromURL {
		dl.FilenameGuessFromURL = ""
	}

	dl.isReady = true

	return dl, nil
}
