package logo

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

// TODO: Add formatting options e.g. alignment, customised formatting

// Formatter is the interface for appender formats.
type Formatter interface {
	Format(l *LogMessage)
	Names() []string
}

type literalFormatter struct {
	s string
}

func (f *literalFormatter) Format(m *LogMessage) {
	m.WriteString(f.s)
}

func (f *literalFormatter) Names() []string {
	return []string{}
}

func newDateFormatter() *dateFormatter {
	return &dateFormatter{buf: newTmpBuffer()}
}

type dateFormatter struct {
	mu  sync.Mutex
	buf *tmpBuffer
}

func (f *dateFormatter) Format(m *LogMessage) {
	// TODO: enable finer granularity for time format
	f.mu.Lock()
	defer f.mu.Unlock()
	// format using logo standard time format:
	f.buf.reset()
	f.setTime(m.timestamp)
	m.Write(f.buf.b[:f.buf.pos])
}

func (f *dateFormatter) setTime(t time.Time) {
	t = t.UTC() // TODO: pull this out into format options
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	micro := t.Nanosecond() / 1000
	// yyyy-mm-dd hh:mm:ss.uuuuuu
	f.buf.padNDigits(int(year), 4)
	f.buf.add('-')
	f.buf.pad2Digits(int(month))
	f.buf.add('-')
	f.buf.pad2Digits(int(day))
	f.buf.add(' ')
	f.buf.pad2Digits(int(hour))
	f.buf.add(':')
	f.buf.pad2Digits(int(minute))
	f.buf.add(':')
	f.buf.pad2Digits(int(second))
	f.buf.add('.')
	f.buf.padNDigits(int(micro), 6)
}

func (f *dateFormatter) Names() []string {
	return []string{"date", "d"}
}

func newTmpBuffer() *tmpBuffer {
	return &tmpBuffer{b: make([]byte, 50)}
}

type tmpBuffer struct {
	b   []byte
	pos int
}

func (t *tmpBuffer) reset() {
	t.pos = 0
}

func (t *tmpBuffer) add(b byte) {
	t.b[t.pos] = b
	t.pos++
}

const digits = "0123456789" // helper to convert int to char

func (t *tmpBuffer) pad2Digits(i int) {
	t.b[t.pos+1] = digits[i%10]
	i /= 10
	t.b[t.pos] = digits[i%10]
	t.pos += 2
}

func (t *tmpBuffer) padNDigits(i, n int) {
	j := n - 1
	for ; j >= 0 && i > 0; j-- {
		t.b[t.pos+j] = digits[i%10]
		i /= 10
	}
	for ; j >= 0; j-- {
		t.b[t.pos+j] = '0'
	}
	t.pos += n
}

type severityFormatter struct{}

func (f *severityFormatter) Format(m *LogMessage) {
	m.WriteString(severityName[m.severity])
}

func (f *severityFormatter) Names() []string {
	return []string{"severity", "s"}
}

type loggerFormatter struct{}

func (f *loggerFormatter) Format(m *LogMessage) {
	m.WriteString(m.name)
}

func (f *loggerFormatter) Names() []string {
	return []string{"logger"}
}

type fileFormatter struct{}

func (f *fileFormatter) Format(m *LogMessage) {
	m.WriteString(m.file)
}

func (f *fileFormatter) Names() []string {
	return []string{"file", "f"}
}

type lineFormatter struct{}

func (f *lineFormatter) Format(m *LogMessage) {
	m.WriteString(strconv.Itoa(m.line))
}

func (f *lineFormatter) Names() []string {
	return []string{"line"}
}

type contextFormatter struct{}

func (f *contextFormatter) Format(m *LogMessage) {
	m.WriteString(m.ctx)
}

func (f *contextFormatter) Names() []string {
	return []string{"context", "c"}
}

type messageFormatter struct{}

func (f *messageFormatter) Format(m *LogMessage) {
	if len(m.format) > 0 {
		fmt.Fprintf(m, m.format, m.args...)
	} else {
		fmt.Fprint(m, m.args...)
	}
}

func (f *messageFormatter) Names() []string {
	return []string{"message", "m"}
}

type newlineFormatter struct{}

func (f *newlineFormatter) Format(m *LogMessage) {
	m.WriteByte('\n')
}

func (f *newlineFormatter) Names() []string {
	return []string{"newline", "n"}
}

var formatters = []Formatter{
	newDateFormatter(),
	&severityFormatter{},
	&loggerFormatter{},
	&fileFormatter{},
	&lineFormatter{},
	&contextFormatter{},
	&messageFormatter{},
	&newlineFormatter{},
}

func extract(format string) ([]Formatter, error) {
	s := []Formatter{}
	p := []byte{}
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			p = append(p, c)
		} else {
			if len(format[i:]) == 1 {
				p = append(p, c)
				continue
			}
			if format[i+1] == '%' { //escaped
				p = append(p, c)
				i++
				continue
			}
			i++

			// save what we have
			if len(p) > 0 {
				l := literalFormatter{s: string(p)}
				s = append(s, &l)
				p = []byte{}
			}

			// now get the formatter
			ok := false
			for _, f := range formatters {
				for _, t := range f.Names() {
					if len(t) <= len(format[i:]) {
						if t == format[i:i+len(t)] {
							s = append(s, f)
							i = i + len(t) - 1
							ok = true
							break
						}
					}
				}
				if ok {
					break
				}
			}

			if !ok {
				return nil, fmt.Errorf("invalid syntax at position %d, %s", i-1, format)
			}
		}

	}
	if len(p) > 0 {
		l := literalFormatter{s: string(p)}
		s = append(s, &l)
	}
	return s, nil
}
