package logo

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEmptyAppenderSetFormatReturnsNoError(t *testing.T) {
	want := error(nil)
	appender := &emptyAppender{}
	err := appender.SetFormat("Futile as never checks or returns error")

	if err != nil {
		got := err.Error()
		t.Errorf("SetFormat got %v, want %v", got, want)
	}
}

func TestEmptyAppenderAppendDoesNothing(t *testing.T) {
	want := 0
	appender := &emptyAppender{}
	m := testMessage()
	appender.Append(m)

	got := len(m.Bytes())
	if got != want {
		t.Errorf("Byte count got %d, want %d", got, want)
	}
}

func TestConsoleAppenderWithDefaultFormat(t *testing.T) {
	want := "2016-04-09 18:03:28.342017 INFO (sample.go:456) - Test 34 (56)\n"

	appender := newConsoleAppender()
	var b bytes.Buffer
	appender.out = &b

	m := testMessage()
	appender.Append(m)
	got := string(b.Bytes())
	if got != want {
		t.Errorf("DefaultFormat got %q, want %q", got, want)
	}
}

func TestConsoleAppenderSetFormat(t *testing.T) {
	want := "2016-04-09 18:03:28.342017-INFOINFO\nLogger (sample.go:456)\n"

	appender := newConsoleAppender()
	var b bytes.Buffer
	appender.out = &b
	appender.SetFormat("%d-%s%s%n%logger (%f:%line)%n")

	m := testMessage()
	appender.Append(m)
	got := string(b.Bytes())
	if got != want {
		t.Errorf("CustomFormat got %q, want %q", got, want)
	}
}

func TestConsoleAppenderSetFormatReturnsErrorWhenInvalidSyntax(t *testing.T) {
	want := "invalid syntax at position 5, bla%%%h blah"

	appender := newConsoleAppender()
	err := appender.SetFormat("bla%%%h blah")
	if err == nil {
		t.Errorf("Error <nil>, want %q", want)
		return
	}
	got := err.Error()
	if got != want {
		t.Errorf("Error got %q, want %q", got, want)
	}
}

func TestRollingFileAppenderWithDefaultFormat(t *testing.T) {
	want := "2016-04-09 18:03:28.342017 INFO (sample.go:456) - Test 34 (56)\n"

	appender := &rollingFileAppender{max: 1024}
	appender.SetFormat(defaultFormat)
	var b bytes.Buffer
	appender.Writer = bufio.NewWriter(&b)

	m := testMessage()
	appender.Append(m)
	appender.Close()
	got := string(b.Bytes())
	if got != want {
		t.Errorf("DefaultFormat got %q, want %q", got, want)
	}
}

func TestRollingFileAppenderSetFormatReturnsErrorWhenInvalidSyntax(t *testing.T) {
	want := "invalid syntax at position 5, bla%%%h blah"

	appender := &rollingFileAppender{}
	err := appender.SetFormat("bla%%%h blah")
	if err == nil {
		t.Errorf("Error <nil>, want %q", want)
		return
	}
	got := err.Error()
	if got != want {
		t.Errorf("Error got %q, want %q", got, want)
	}
}

func TestRollingFileAppenderTracksBytesWritten(t *testing.T) {
	l := len("2016-04-09 18:03:28.342017 INFO (sample.go:456) - Test 34 (56)\n")
	want := uint64(l)

	appender := &rollingFileAppender{max: 1024}
	appender.SetFormat(defaultFormat)
	var b bytes.Buffer
	appender.Writer = bufio.NewWriter(&b)

	m := testMessage()
	appender.Append(m)
	got := appender.bytes
	if got != want {
		t.Errorf("Bytes written got %d, want %d", got, want)
	}
}

func TestLogNameAddsDateToFilename(t *testing.T) {
	want := "test.log.20161119-151415."
	timenow = func() time.Time {
		t, _ := time.Parse("2006-01-02T15:04:05.999999999", "2016-11-19T15:14:15.123456789")
		return t
	}
	defer reset()

	filename := "test.log"

	l := logname(filename)

	got := strings.TrimSuffix(l, strconv.Itoa(pid))

	if got != want {
		t.Errorf("Logname got %q, want %q", got, want)
	}
}

func TestTestAppenderWithDefaultFormat(t *testing.T) {
	want := "2016-04-09 18:03:28.342017 INFO (sample.go:456) - Test 34 (56)\n"

	appender := TestAppender
	m := testMessage()
	appender.Append(m)
	msgs := appender.Messages
	if len(msgs) != 1 {
		t.Errorf("Message count got %d, want 1", len(msgs))
		return
	}
	got := msgs[0]
	if got != want {
		t.Errorf("DefaultFormat got %q, want %q", got, want)
	}
}

func TestTestAppenderSetFormatMessage(t *testing.T) {
	want := "2016-04-09 18:03:28.342017-INFOINFO\nLogger (sample.go:456)\n"

	appender := newTestAppender()
	appender.SetFormat("%d-%s%s%n%logger (%f:%line)%n")
	m := testMessage()
	appender.Append(m)
	msgs := appender.Messages
	if len(msgs) != 1 {
		t.Errorf("Message count got %d, want 1", len(msgs))
		return
	}
	got := msgs[0]
	if got != want {
		t.Errorf("CustomFormat got %q, want %q", got, want)
	}
}

func TestTestAppenderSetFormatStoresFormat(t *testing.T) {
	want := "%d-%s%s%n%logger (%f:%line)%n"

	appender := newTestAppender()
	appender.SetFormat("%d-%s%s%n%logger (%f:%line)%n")
	got := appender.Format
	if got != want {
		t.Errorf("Format got %q, want %q", got, want)
	}
}

func TestTestAppenderSetFormatReturnsErrorWhenInvalidSyntax(t *testing.T) {
	want := "invalid syntax at position 5, bla%%%h blah"

	appender := newTestAppender()
	err := appender.SetFormat("bla%%%h blah")
	if err == nil {
		t.Errorf("Error <nil>, want %q", want)
		return
	}
	got := err.Error()
	if got != want {
		t.Errorf("Error got %q, want %q", got, want)
	}
}

func TestTestAppenderCloseSetsClosed(t *testing.T) {
	want := true

	appender := newTestAppender()
	if appender.Closed {
		t.Errorf("Appender already closed")
		return
	}
	appender.Close()
	got := appender.Closed
	if got != want {
		t.Errorf("Closed got %t, want %t", got, want)
	}
}

func TestTestAppenderReset(t *testing.T) {
	appender := newTestAppender()
	appender.SetFormat("%d-%s%s%n%logger (%f:%line)%n")
	m := testMessage()
	appender.Append(m)
	appender.Close()

	appender.Reset()

	var tests = []struct {
		property string
		f        func() interface{}
		want     interface{}
	}{
		{"Closed", func() interface{} { return appender.Closed }, false},
		{"Format", func() interface{} { return appender.Format }, defaultFormat},
		{"Messages count", func() interface{} { return len(appender.Messages) }, 0},
		{"LogMessages count", func() interface{} { return len(appender.logMessages) }, 0},
	}

	for _, test := range tests {
		got := test.f()
		if got != test.want {
			t.Errorf("%s got %v, want %v", test.property, got, test.want)
		}
	}
}
