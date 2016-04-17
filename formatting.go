package logo

import (
	"fmt"
	"strconv"
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

type dateFormatter struct{}

func (f *dateFormatter) Format(m *LogMessage) {
	m.Write(m.timestamp)
}

func (f *dateFormatter) Names() []string {
	return []string{"date", "d"}
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
	&dateFormatter{},
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
	p := "" // TODO: make into []byte
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			p += string(c)
		} else {
			if len(format[i:]) == 1 {
				p += string(c)
				continue
			}
			if format[i+1] == '%' { //escaped
				p += string(c)
				i++
				continue
			}
			i++

			// save what we have
			if len(p) > 0 {
				l := literalFormatter{s: p}
				s = append(s, &l)
				p = ""
			}
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
		l := literalFormatter{s: p}
		s = append(s, &l)
	}
	return s, nil
}
