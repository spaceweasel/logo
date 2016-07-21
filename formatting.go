package logo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TODO: Add formatting options e.g. alignment, customised formatting

// Formatter is the interface for appender formats.
type Formatter interface {
	Format(l *LogMessage)
	Names() []string
	WithParameter(p string) Formatter
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

func (f *literalFormatter) WithParameter(p string) Formatter {
	return &literalFormatter{s: p}
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

func (f *dateFormatter) WithParameter(p string) Formatter {
	return newDateFormatter()
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

func (f *severityFormatter) WithParameter(p string) Formatter {
	return &severityFormatter{}
}

type loggerFormatter struct{}

func (f *loggerFormatter) Format(m *LogMessage) {
	m.WriteString(m.name)
}

func (f *loggerFormatter) Names() []string {
	return []string{"logger"}
}

func (f *loggerFormatter) WithParameter(p string) Formatter {
	return &loggerFormatter{}
}

type fileFormatter struct{}

func (f *fileFormatter) Format(m *LogMessage) {
	m.WriteString(m.file)
}

func (f *fileFormatter) Names() []string {
	return []string{"file", "f"}
}

func (f *fileFormatter) WithParameter(p string) Formatter {
	return &fileFormatter{}
}

type lineFormatter struct{}

func (f *lineFormatter) Format(m *LogMessage) {
	m.WriteString(strconv.Itoa(m.line))
}

func (f *lineFormatter) Names() []string {
	return []string{"line"}
}

func (f *lineFormatter) WithParameter(p string) Formatter {
	return &lineFormatter{}
}

// Deprecated: contextFormatter is deprecated - use propertyFormatter instead.
type contextFormatter struct{}

func (f *contextFormatter) Format(m *LogMessage) {
	m.WriteString(m.ctx)
}

func (f *contextFormatter) Names() []string {
	return []string{"context", "c"}
}

func (f *contextFormatter) WithParameter(p string) Formatter {
	return &contextFormatter{}
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

func (f *messageFormatter) WithParameter(p string) Formatter {
	return &messageFormatter{}
}

type newlineFormatter struct{}

func (f *newlineFormatter) Format(m *LogMessage) {
	m.WriteByte('\n')
}

func (f *newlineFormatter) Names() []string {
	return []string{"newline", "n"}
}

func (f *newlineFormatter) WithParameter(p string) Formatter {
	return &newlineFormatter{}
}

type propertyFormatter struct {
	name string
}

func (f *propertyFormatter) Format(m *LogMessage) {
	v, ok := m.properties[f.name]
	if !ok {
		return
	}

	m.WriteString(fmt.Sprint(v))
}

func (f *propertyFormatter) Names() []string {
	return []string{"property", "p"}
}

func (f *propertyFormatter) WithParameter(p string) Formatter {
	return &propertyFormatter{name: p}
}

type jsonFormatter struct{}

func (f *jsonFormatter) Format(m *LogMessage) {
	d := make(map[string]interface{})
	d["@timestamp"] = m.timestamp
	d["@version"] = "1"
	d["level"] = severityName[m.severity]
	d["level_value"] = m.severity
	d["logger_name"] = m.name
	d["file"] = m.file
	d["line"] = m.line

	for k, v := range m.properties {
		d[k] = v
	}

	if len(m.format) > 0 {
		d["message"] = fmt.Sprintf(m.format, m.args...)
	} else {
		if len(m.args) == 1 {
			d["message"] = m.args[0]
		} else {
			d["message"] = m.args
		}
	}

	b, err := json.Marshal(d)
	if err != nil {
		return
	}

	m.Write(b)
}

func (f *jsonFormatter) Names() []string {
	return []string{"JSON"}
}

func (f *jsonFormatter) WithParameter(p string) Formatter {
	// TODO: Consider indent option
	return &jsonFormatter{}
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
	&propertyFormatter{},
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
							i = i + len(t)
							if i < len(format) && format[i] == '{' {
								j := strings.IndexByte(format[i:], byte('}'))
								if j == -1 {
									return nil, fmt.Errorf("invalid syntax - unclosed parameter brace at position %d, %s", i, format)
								}
								name := format[i+1 : i+j]
								i = i + j + 1
								f = f.WithParameter(name)
							}
							i--
							s = append(s, f)
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
