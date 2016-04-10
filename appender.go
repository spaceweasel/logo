package logo

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// TODO: Add severity filters to appenders

//var defaultFormat = "%date %severity %logger - %message%newline"
var defaultFormat = "%date %severity (%file:%line) - %message%newline"

type Appender interface {
	Append(m *LogMessage)
	SetFormat(format string) error
	Close()
}

var EmptyAppender = &emptyAppender{}

type emptyAppender struct{}

func (a *emptyAppender) Append(m *LogMessage) {}

func (a *emptyAppender) Close() {}

func (a *emptyAppender) SetFormat(format string) error { return nil }

var ConsoleAppender = newConsoleAppender()

func newConsoleAppender() *consoleAppender {
	a := consoleAppender{out: os.Stderr}
	a.SetFormat(defaultFormat)
	return &a
}

type consoleAppender struct {
	mu         sync.Mutex
	out        io.Writer
	formatters []Formatter
}

func (a *consoleAppender) SetFormat(format string) error {
	f, err := extract(format)
	if err != nil {
		return err
	}
	a.formatters = f
	return nil
}

func (a *consoleAppender) Append(m *LogMessage) {
	m.Reset()
	for _, f := range a.formatters {
		f.Format(m)
	}
	a.Write(m.Bytes())
}

func (a *consoleAppender) Write(p []byte) (n int, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.out.Write(p)
}

func (a *consoleAppender) Close() {
	// to satisfy interface
}

type rollingFileAppender struct {
	*bufio.Writer
	mu       sync.Mutex
	filename string
	file     *os.File
	// TODO: split filename into dir & file parts
	//directory *string
	bytes      uint64
	max        uint64
	formatters []Formatter
}

func RollingFileAppender(filename string, max uint64) (Appender, error) {
	a := rollingFileAppender{
		filename: filename,
		max:      max * 1024 * 1024, // megabytes
	}
	a.SetFormat(defaultFormat)
	// if a.directory == nil {
	// 	dir := filepath.Join(os.TempDir(), filepath.Base(os.Args[0]))
	// 	a.directory = &dir
	// }
	err := a.rotate()
	if err != nil {
		return nil, err
	}
	go a.flusher()
	return &a, nil
}

func (a *rollingFileAppender) SetFormat(format string) error {
	f, err := extract(format)
	if err != nil {
		return err
	}
	a.formatters = f
	return nil
}

func (a *rollingFileAppender) Append(m *LogMessage) {
	m.Reset()
	for _, f := range a.formatters {
		f.Format(m)
	}
	a.Write(m.Bytes())
}

func (a *rollingFileAppender) Write(p []byte) (n int, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.bytes+uint64(len(p)) >= a.max {
		// TODO: consider what to do with any error from rotate().
		// Just ignore and continue or exit the program?
		a.rotate()
	}
	n, err = a.Writer.Write(p)
	a.bytes += uint64(n)
	return
}

const flushInterval = 30 * time.Second

func (a *rollingFileAppender) flusher() {
	for _ = range time.NewTicker(flushInterval).C {
		a.mu.Lock()
		if a.Writer != nil {
			a.Flush()
			a.file.Sync()
		}
		a.mu.Unlock()
	}
}

const bufferSize = 256 * 1024

func (a *rollingFileAppender) rotate() error {
	if a.file != nil {
		a.Close()
	}
	a.bytes = 0
	name := logname(a.filename)
	var err error
	a.file, err = os.Create(name)
	if err != nil {
		return err
	}
	//a.Writer = bufio.NewWriter(a.file)	// default size is 4096
	a.Writer = bufio.NewWriterSize(a.file, bufferSize)
	//n, err := a.file.WriteString("New log created!\n")
	//a.bytes = uint64(n)
	return nil
}

var pid = os.Getpid()

func logname(fname string) string {
	t := timenow().UTC()
	return fmt.Sprintf("%s.%04d%02d%02d-%02d%02d%02d.%d",
		fname,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		pid)
}

func (a *rollingFileAppender) Close() {
	if a.Writer != nil {
		a.Flush()
		a.file.Close()
		a.Writer = nil
		a.file = nil
	}
}
