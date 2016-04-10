package logo

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

type severity int

const (
	debug severity = iota
	info
	warn
	errorMsg
	panicMsg
	fatal
)

var severityName = []string{
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
	"PANIC",
	"FATAL",
}

func severityFromName(n string) severity {
	n = strings.ToUpper(n)
	for i, s := range severityName {
		if s == n {
			return severity(i)
		}
	}
	return severity(0)
}

type logManager struct {
	appenders map[string]Appender
	// TODO: expose these level properties
	level       severity
	manageLevel bool
}

var manager = newLogManager()
var defaultLogger = newDefaultLogger()

func newLogManager() *logManager {
	m := logManager{appenders: make(map[string]Appender)}
	m.appenders["console"] = ConsoleAppender
	return &m
}

func newDefaultLogger() *Logger {
	l := New("", "debug")
	l.callDepth = 3
	return l
}

// func (l *logManager) Close() {
// 	for _, a := range l.appenders {
// 		a.Close()
// 	}
// }

func Close() {
	for _, a := range manager.appenders {
		a.Close()
	}
}

func AddAppender(name string, a Appender) {
	manager.appenders[name] = a
}

// func (l *logManager) AddAppender(name string, a Appender) {
// 	l.appenders[name] = a
// }

type LogMessage struct {
	bytes.Buffer
	format    string
	args      []interface{}
	severity  string
	name      string
	file      string
	line      int
	ctx       string
	timestamp []byte
}

func (l *LogMessage) setTime(t time.Time) {
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	micro := t.Nanosecond() / 1000
	// yyyy-mm-dd hh:mm:ss.uuuuuu
	padw(l.timestamp, 0, int(year), 4)
	l.timestamp[4] = '-'
	pad2(l.timestamp, 5, int(month))
	l.timestamp[7] = '-'
	pad2(l.timestamp, 8, int(day))
	l.timestamp[10] = ' '
	pad2(l.timestamp, 11, int(hour))
	l.timestamp[13] = ':'
	pad2(l.timestamp, 14, int(minute))
	l.timestamp[16] = ':'
	pad2(l.timestamp, 17, int(second))
	l.timestamp[19] = '.'
	padw(l.timestamp, 20, int(micro), 6)
}

var timenow = time.Now

func (l *logManager) output(callDepth int, appenders []Appender, s severity, name, ctx string, format string, v ...interface{}) {
	if manager.manageLevel && s < manager.level {
		return
	}

	// TODO: get these from a pool
	msg := &LogMessage{
		severity:  severityName[s],
		name:      name,
		ctx:       ctx,
		args:      v,
		format:    format,
		timestamp: make([]byte, 26),
	}

	_, file, line, ok := runtime.Caller(callDepth)
	if !ok {
		file = "???"
		line = 0
	} else {
		slash := strings.LastIndex(file, "/")
		if slash >= 0 {
			file = file[slash+1:]
		}
	}
	msg.file = file
	msg.line = line
	msg.setTime(timenow().UTC())
	for _, a := range appenders {
		a.Append(msg)
	}
}

const digits = "0123456789" // helper to convert int to char

func pad2(buf []byte, i, n int) {
	buf[i+1] = digits[n%10]
	n /= 10
	buf[i] = digits[n%10]
}

func padw(buf []byte, i, n, w int) {
	j := w - 1
	for ; j >= 0 && n > 0; j-- {
		buf[i+j] = digits[n%10]
		n /= 10
	}
	for ; j >= 0; j-- {
		buf[i+j] = '0'
	}
}

func New(name string, level string) *Logger {
	sev := severityFromName(level)
	return &Logger{
		level: sev,
		name:  name,
		a:     []Appender{ConsoleAppender},
		//lm:        manager,
		callDepth: 2,
	}
}

// func (l *logManager) NewLogger(name string, level string) *Logger {
// 	sev := severityFromName(level)
// 	if l.manageLevel && sev < l.level {
// 		sev = l.level
// 	}
// 	return &Logger{
// 		level:     sev,
// 		name:      name,
// 		a:         []Appender{ConsoleAppender},
// 		lm:        l,
// 		callDepth: 2,
// 	}
// }

type Logger struct {
	lm        *logManager
	level     severity
	name      string
	a         []Appender
	context   string
	callDepth int
}

func (l *Logger) SetAppenders(names ...string) {
	l.a = []Appender{}
	for _, n := range names {
		if a, ok := manager.appenders[n]; !ok {
			panic(fmt.Sprintf("unrecognised appender, [%s]", n))
		} else {
			l.a = append(l.a, a)
		}
	}
}

func (l *Logger) WithContext(context fmt.Stringer) *Logger {
	return &Logger{
		//lm:      manager,
		level:   l.level,
		name:    l.name,
		a:       l.a,
		context: context.String(),
	}
}

func (l *Logger) Debug(v ...interface{}) {
	if l.level > debug {
		return
	}
	manager.output(l.callDepth, l.a, debug, l.name, l.context, "", v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.level > debug {
		return
	}
	manager.output(l.callDepth, l.a, debug, l.name, l.context, format, v...)
}

func (l *Logger) Info(v ...interface{}) {
	if l.level > info {
		return
	}
	manager.output(l.callDepth, l.a, info, l.name, l.context, "", v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	if l.level > info {
		return
	}
	manager.output(l.callDepth, l.a, info, l.name, l.context, format, v...)
}

func (l *Logger) Warn(v ...interface{}) {
	if l.level > warn {
		return
	}
	manager.output(l.callDepth, l.a, warn, l.name, l.context, "", v...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.level > warn {
		return
	}
	manager.output(l.callDepth, l.a, warn, l.name, l.context, format, v...)
}

func (l *Logger) Error(v ...interface{}) {
	if l.level > errorMsg {
		return
	}
	manager.output(l.callDepth, l.a, errorMsg, l.name, l.context, "", v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.level > errorMsg {
		return
	}
	manager.output(l.callDepth, l.a, errorMsg, l.name, l.context, format, v...)
}

func (l *Logger) Panic(v ...interface{}) {
	if l.level <= panicMsg {
		manager.output(l.callDepth, l.a, panicMsg, l.name, l.context, "", v...)
	}
	panic(fmt.Sprint(v...))
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	if l.level <= panicMsg {
		manager.output(l.callDepth, l.a, panicMsg, l.name, l.context, format, v...)
	}
	panic(fmt.Sprintf(format, v...))
}

var exit = func(i int) { os.Exit(i) }

func (l *Logger) Fatal(v ...interface{}) {
	manager.output(l.callDepth, l.a, fatal, l.name, l.context, "", v...)
	Close()
	exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	manager.output(l.callDepth, l.a, fatal, l.name, l.context, format, v...)
	Close()
	exit(1)
}

func SetAppenders(names ...string) {
	defaultLogger.SetAppenders(names...)
}

func Debugf(format string, v ...interface{}) {
	defaultLogger.Debugf(format, v...)
}

func Debug(v ...interface{}) {
	defaultLogger.Debug(v...)
}

func Infof(format string, v ...interface{}) {
	defaultLogger.Infof(format, v...)
}

func Info(v ...interface{}) {
	defaultLogger.Info(v...)
}

func Warnf(format string, v ...interface{}) {
	defaultLogger.Warnf(format, v...)
}

func Warn(v ...interface{}) {
	defaultLogger.Warn(v...)
}

func Errorf(format string, v ...interface{}) {
	defaultLogger.Errorf(format, v...)
}

func Error(v ...interface{}) {
	defaultLogger.Error(v...)
}

func Panicf(format string, v ...interface{}) {
	defaultLogger.Panicf(format, v...)
}

func Panic(v ...interface{}) {
	defaultLogger.Panic(v...)
}

func Fatalf(format string, v ...interface{}) {
	defaultLogger.Fatalf(format, v...)
}

func Fatal(v ...interface{}) {
	defaultLogger.Fatal(v...)
}
