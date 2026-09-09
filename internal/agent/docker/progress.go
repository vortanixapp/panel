package docker

import (
	"io"
	"strings"
	"sync"
	"time"
)

type ProgressFunc func(stage string, percent int, message, line string)

const (
	StageDownload = "download"
	StageExtract  = "extract"
	StageSteamCMD = "steamcmd"
	StageDone     = "done"
)

func noopProgress(string, int, string, string) {}

type progressReader struct {
	r        io.Reader
	total    int64
	read     int64
	report   ProgressFunc
	lastSent time.Time
	lastPct  int
}

func newProgressReader(r io.Reader, total int64, report ProgressFunc) *progressReader {
	if report == nil {
		report = noopProgress
	}
	return &progressReader{r: r, total: total, report: report}
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.read += int64(n)

	if p.total > 0 && time.Since(p.lastSent) >= time.Second {
		pct := int(float64(p.read) * 100 / float64(p.total))
		if pct > 100 {
			pct = 100
		}
		if pct != p.lastPct {
			p.lastPct = pct
			p.lastSent = time.Now()
			p.report(StageDownload, pct/2, "Скачивание файлов сервера", "")
		}
	}
	return n, err
}

type lineScanner struct {
	mu     sync.Mutex
	buf    strings.Builder
	report func(line string)
}

func newLineScanner(report func(line string)) *lineScanner {
	return &lineScanner{report: report}
}

func (l *lineScanner) Write(b []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, c := range string(b) {
		if c == '\n' || c == '\r' {
			l.flushLocked()
			continue
		}
		l.buf.WriteRune(c)
	}
	if l.buf.Len() > 512 {
		l.flushLocked()
	}
	return len(b), nil
}

func (l *lineScanner) flushLocked() {
	line := strings.TrimSpace(l.buf.String())
	l.buf.Reset()
	if line != "" && l.report != nil {
		l.report(line)
	}
}

func (l *lineScanner) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flushLocked()
	return nil
}

func steamPercent(line string) int {
	idx := strings.Index(line, "progress:")
	if idx < 0 {
		return -1
	}
	rest := strings.TrimSpace(line[idx+len("progress:"):])
	end := strings.IndexAny(rest, " (")
	if end > 0 {
		rest = rest[:end]
	}
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		rest = rest[:dot]
	}
	pct := 0
	for _, c := range rest {
		if c < '0' || c > '9' {
			return -1
		}
		pct = pct*10 + int(c-'0')
	}
	if pct > 100 {
		return 100
	}
	return pct
}
