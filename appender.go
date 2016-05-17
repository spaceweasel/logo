package logo

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var defaultFormat = "%date %severity (%file:%line) - %message%newline"

// Appender is the interface describing a logger appender.
// Each logger may have one or more appenders and will call Append
// passing in a LogMessage. Append will format the message using
// the native format for the logger (normally defaultformat), or a
// custom format specified with SetFormat. Custom format strings comprise
// one or more percent tags. For example, %severity will include the
// log message severity. Many tags have shorthand too, e.g. %s has the
// same effect as %severity.
// SetFilters adds another level of granularity, restricting logging to
// messages which have a severity in the accepted list. For example, to
// restrict an appender to log only debug and warning messages:
//  appender.SetFilters("debug", "warn")
// Close is called when the log manager is closed; Appender implementations
// must use this to flush and close any open files, connections, etc.
type Appender interface {
	Append(m *LogMessage)
	SetFormat(format string) error
	Close()
	SetFilters(f ...string)
}

// EmptyAppender is used for testing.
// All methods succeed, but perform no action.
var EmptyAppender = &emptyAppender{}

type emptyAppender struct{}

func (a *emptyAppender) Append(m *LogMessage) {}

func (a *emptyAppender) Close() {}

func (a *emptyAppender) SetFormat(format string) error { return nil }

func (a *emptyAppender) SetFilters(f ...string) {}

// ConsoleAppender writes formatted log messages to StdErr using the default format.
var ConsoleAppender = newConsoleAppender()

func newConsoleAppender() *consoleAppender {
	a := consoleAppender{
		out: os.Stderr,
	}
	a.SetFormat(defaultFormat)
	a.SetFilters(severityName...)
	return &a
}

type consoleAppender struct {
	mu         sync.Mutex
	out        io.Writer
	formatters []Formatter
	filters    map[severity]bool
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
	if !a.filters[m.severity] {
		return
	}
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

func (a *consoleAppender) SetFilters(f ...string) {
	a.filters = make(map[severity]bool)
	for _, n := range f {
		s := severityFromName(n)
		a.filters[s] = true
	}
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
	filters    map[severity]bool
}

// RollingFileAppender returns a new rollingfile appender instance.
// RollingFileAppender writes formatted log messages to the specified file.
// Filename must include the full path and logfile name. If only the file
// name is supplied, logging will be to the current directory.
// RollingFileAppender will create a new file each time max MB have been
// written. New files are created with a date-time based suffix, but old files
// are not deleted.
// Note that RollingFileAppender will always create a new file, and never appends
// to an exiting file. If an application is started and stopped quickly several
// times, then this will result in the creation of the same number of log files;
// even though max bytes may not have been written to any of them.
// RollingFileAppender buffers messages to improve performance and reduce
// blocking. Buffered data is written to disk every 30 seconds and when Close
// is called.
// RollingFileAppender uses the default format.
func RollingFileAppender(filename string, max int) (Appender, error) {
	m := uint64(max) * 1024 * 1024 // megabytes
	a := rollingFileAppender{
		filename: filename,
		max:      m,
	}
	a.SetFormat(defaultFormat)
	a.SetFilters(severityName...)
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

func (a *rollingFileAppender) SetFilters(f ...string) {
	a.filters = make(map[severity]bool)
	for _, n := range f {
		s := severityFromName(n)
		a.filters[s] = true
	}
}

func (a *rollingFileAppender) Append(m *LogMessage) {
	m.Reset()
	if !a.filters[m.severity] {
		return
	}
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
		// TODO: Call a LogRotateError() stub
		a.rotate()
	}
	n, err = a.Writer.Write(p)
	// TODO: Keep track of number of bytes written since last flush
	// and force if 95% of max?
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

// TestAppender is used for testing.
// Applications can specify this appender and verify logger calls
// by examining the properties: Messages, Format and Closed.
var TestAppender = newTestAppender()

func newTestAppender() *testAppender {
	b := new(bytes.Buffer)
	a := testAppender{
		buf:             b,
		consoleAppender: &consoleAppender{out: b},
	}
	a.SetFormat(defaultFormat)
	a.SetFilters(severityName...)
	return &a
}

type testAppender struct {
	*consoleAppender
	logMessages []*LogMessage
	Closed      bool
	Format      string
	Messages    []string
	buf         *bytes.Buffer
}

func (a *testAppender) Append(m *LogMessage) {
	n := &LogMessage{
		format:    m.format,
		severity:  m.severity,
		name:      m.name,
		file:      m.file,
		line:      m.line,
		ctx:       m.ctx,
		timestamp: m.timestamp,
	}
	for _, a := range m.args {
		n.args = append(n.args, a)
	}

	a.logMessages = append(a.logMessages, n)
	a.consoleAppender.Append(m)
	s := string(a.buf.Bytes())
	a.Messages = append(a.Messages, s)
}

func (a *testAppender) Close() {
	a.consoleAppender.Close()
	a.Closed = true
}

func (a *testAppender) SetFormat(format string) error {
	a.Format = format
	return a.consoleAppender.SetFormat(format)
}

func (a *testAppender) SetFilters(f ...string) {
	a.consoleAppender.SetFilters(f...)
}

func (a *testAppender) Reset() {
	a.logMessages = []*LogMessage{}
	a.Messages = []string{}
	a.Closed = false
	a.buf.Reset()
	a.SetFormat(defaultFormat)
}
