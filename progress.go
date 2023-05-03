package main

import (
	"fmt"
	"math"
	"time"

	humanize "github.com/dustin/go-humanize"
)

type progressBar struct {
	total     int64
	current   int64
	snapRate  int64
	snapTime  time.Time
	snapBytes int64
	emit      func(string, interface{})
	next      time.Time
	pct       float32
	status    string
}

func (p *progressBar) round(x float64) int64 {
	t := math.Trunc(x)
	if math.Abs(x-t) >= 0.5 {
		return int64(t + math.Copysign(1, x))
	}
	return int64(t)
}

func (p *progressBar) Write(b []byte) (int, error) {
	l := len(b)
	p.current += int64(l)
	p.update()
	return l, nil
}

func (p *progressBar) update() {
	if time.Now().After(p.next) {
		p.next = time.Now().Add(250 * time.Millisecond)
		pct := float32(p.current) / float32(p.total) * 100
		now := time.Now()
		since := time.Now().Sub(p.snapTime)
		if since > time.Second {
			p.snapRate = p.round(float64(p.current-p.snapBytes) / since.Seconds())
			p.snapTime = now
			p.snapBytes = p.current
		}
		p.pct = pct
		p.status = fmt.Sprintf(
			"%2.2f%% Downloaded: %s of %s (%s/sec)",
			pct,
			humanize.Bytes(uint64(p.current)),
			humanize.Bytes(uint64(p.total)),
			humanize.Bytes(uint64(p.snapRate)),
		)
	}
}
