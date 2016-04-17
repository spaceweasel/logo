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
	// TODO: Should this return an error instead?
	return severity(0)
}

type logManager struct {
	appenders map[string]Appender
	level     severity
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

// Close closes all appenders in the log manager.
// It is important that Close is called before exiting an application
// to ensure that any buffered data is written.
func Close() {
	for _, a := range manager.appenders {
		a.Close()
	}
}

// AddAppender adds a named appender to the log manager.
// Returns an error if an appender of the same name has been
// added previously.
func AddAppender(name string, a Appender) error {
	if _, ok := manager.appenders[name]; ok {
		return fmt.Errorf("appender already exist")
	}
	manager.appenders[name] = a
	return nil
}

// LogMessage is the structure passed to each appender of a logger.
type LogMessage struct {
	bytes.Buffer
	format    string
	args      []interface{}
	severity  severity
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

// SetManagerLevel sets the minimum severity level for logging.
// This affects all managed loggers, regardless of their individual setting.
// For example, if the Warn() method is called on a logger with severity level "info",
// but the manager level is set to "error", no logging occurs.
// The default setting is "debug", which won't restrict any logging.
func SetManagerLevel(level string) {
	sev := severityFromName(level)
	manager.level = sev
}

var timenow = time.Now // to facilitate testing

var pool = make(chan *LogMessage, 50)

func getMessage() *LogMessage {
	var m *LogMessage
	select {
	case m = <-pool:
		m.Reset()
	default:
		m = &LogMessage{
			timestamp: make([]byte, 26),
		}
	}
	return m
}

func putMessage(m *LogMessage) {
	// ditch large buffers
	if m.Len() >= 250 {
		return
	}
	select {
	case pool <- m:
	default: // pool full - continue
	}
}

func (l *logManager) output(callDepth int, appenders []Appender, s severity, name, ctx string, format string, v ...interface{}) {
	if manager.level > debug && s < manager.level {
		return
	}

	msg := getMessage()
	msg.severity = s
	msg.name = name
	msg.ctx = ctx
	msg.args = v
	msg.format = format

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
	putMessage(msg)
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

// New returns a new logger instance.
// Name can be any string value and is included in any log output
// (if specified in the appender format string). Normally this would be
// the name of a package using the logger. Level specifies the minimum
// severity for successful logging.
// For example, if the Warn() or Info() methods are called on a logger
// with severity level "info", then logging will be successful, but calls
// to Debug() will not.
func New(name string, level string) *Logger {
	sev := severityFromName(level)
	return &Logger{
		level:     sev,
		name:      name,
		a:         []Appender{ConsoleAppender},
		callDepth: 2,
	}
}

// Logger is a named logger owned by the log manager. The log manager has
// at least one default logger instance, but additional named loggers can
// be created with the New() method.
type Logger struct {
	lm        *logManager
	level     severity
	name      string
	a         []Appender
	context   string
	callDepth int
}

// SetAppenders specifies one or more appenders that the logger should use.
// Appenders are specified by their string name, and must have been added to
// the log manager previously using the AddAppender method.
// SetAppenders will panic if an appender name is not recognised.
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

// WithContext returns a new context logger instance. The context logger
// is identical to its parent, with the exception that the context property
// is set and will appear in log messages (if specified in the appender
// format). Its primary purpose is for situations where you have a service
// which uses a named logger for general purpose logging, but also needs to
// log some messages with user related data (e.g. an HTTP correlationID
// header).
// The context parameter can be any type, but must implement the fmt.Stringer
// interface, as this will dictate its format in the log message.
func (l *Logger) WithContext(context fmt.Stringer) *Logger {
	return &Logger{
		level:   l.level,
		name:    l.name,
		a:       l.a,
		context: context.String(),
	}
}

// Debug logs with a severity of "debug". Logging only succeeds if both
// the logger level (and manager level) are set to "debug".
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Debug(args ...interface{}) {
	if l.level > debug {
		return
	}
	manager.output(l.callDepth, l.a, debug, l.name, l.context, "", args...)
}

// Debugf logs with a severity of "debug". Logging only succeeds if both
// the logger level (and manager level) are set to "debug".
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.level > debug {
		return
	}
	manager.output(l.callDepth, l.a, debug, l.name, l.context, format, args...)
}

// Info logs with a severity of "info". Logging only succeeds if both the
// logger level (and manager level) are set to "info" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Info(args ...interface{}) {
	if l.level > info {
		return
	}
	manager.output(l.callDepth, l.a, info, l.name, l.context, "", args...)
}

// Infof logs with a severity of "info". Logging only succeeds if both the
// logger level (and manager level) are set to "info" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.level > info {
		return
	}
	manager.output(l.callDepth, l.a, info, l.name, l.context, format, args...)
}

// Warn logs with a severity of "warn". Logging only succeeds if both the
// logger level (and manager level) are set to "warn" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Warn(args ...interface{}) {
	if l.level > warn {
		return
	}
	manager.output(l.callDepth, l.a, warn, l.name, l.context, "", args...)
}

// Warnf logs with a severity of "warn". Logging only succeeds if both the
// logger level (and manager level) are set to "warn" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.level > warn {
		return
	}
	manager.output(l.callDepth, l.a, warn, l.name, l.context, format, args...)
}

// Error logs with a severity of "error". Logging only succeeds if both the
// logger level (and manager level) are set to "error" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Error(args ...interface{}) {
	if l.level > errorMsg {
		return
	}
	manager.output(l.callDepth, l.a, errorMsg, l.name, l.context, "", args...)
}

// Errorf logs with a severity of "error". Logging only succeeds if both the
// logger level (and manager level) are set to "error" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.level > errorMsg {
		return
	}
	manager.output(l.callDepth, l.a, errorMsg, l.name, l.context, format, args...)
}

// Panic logs with a severity of "panic", and then panics with the
// supplied parameters. Logging only succeeds if both the
// logger level (and manager level) are set to "panic" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Panic(args ...interface{}) {
	if l.level <= panicMsg {
		manager.output(l.callDepth, l.a, panicMsg, l.name, l.context, "", args...)
	}
	panic(fmt.Sprint(args...))
}

// Panicf logs with a severity of "panic", and then panics with the
// supplied parameters. Logging only succeeds if both the
// logger level (and manager level) are set to "panic" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Panicf(format string, args ...interface{}) {
	if l.level <= panicMsg {
		manager.output(l.callDepth, l.a, panicMsg, l.name, l.context, format, args...)
	}
	panic(fmt.Sprintf(format, args...))
}

var exit = func(i int) { os.Exit(i) } // to facilitate testing

// Fatal logs with a severity of "fatal", and then calls Exit.
// Logging only succeeds if both the logger level (and manager level)
// are set to "fatal" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Fatal(args ...interface{}) {
	manager.output(l.callDepth, l.a, fatal, l.name, l.context, "", args...)
	Close()
	exit(1)
}

// Fatalf logs with a severity of "fatal", and then calls Exit.
// Logging only succeeds if both the logger level (and manager level)
// are set to "fatal" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	manager.output(l.callDepth, l.a, fatal, l.name, l.context, format, args...)
	Close()
	exit(1)
}

// SetAppenders specifies one or more appenders for the default logger.
// Appenders are specified by their string name, and must have been added to
// the log manager previously using the AddAppender method.
// SetAppenders will panic if an appender name is not recognised.
func SetAppenders(names ...string) {
	defaultLogger.SetAppenders(names...)
}

// Debugf logs to the default logger with a severity of "debug".
// Logging only succeeds if the manager level is set to "debug".
// Arguments are handled in the same manner as fmt.Printf.
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Debug logs to the default logger with a severity of "debug".
// Logging only succeeds if the manager level is set to "debug".
// Arguments are handled in the same manner as fmt.Println.
func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

// Infof logs to the default logger with a severity of "info".
// Logging only succeeds if the manager level is set to "info" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Info logs to the default logger with a severity of "info".
// Logging only succeeds if the manager level is set to "info" or lower.
// Arguments are handled in the same manner as fmt.Println.
func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

// Warnf logs to the default logger with a severity of "warn".
// Logging only succeeds if the manager level is set to "warn" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Warn logs to the default logger with a severity of "warn".
// Logging only succeeds if the manager level is set to "warn" or lower.
// Arguments are handled in the same manner as fmt.Println.
func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

// Errorf logs to the default logger with a severity of "error".
// Logging only succeeds if the manager level is set to "error" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Error logs to the default logger with a severity of "error".
// Logging only succeeds if the manager level is set to "error" or lower.
// Arguments are handled in the same manner as fmt.Println.
func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

// Panicf logs to the default logger with a severity of "panic", and then
// panics with the supplied parameters. Logging only succeeds if the manager
// level is set to "panic" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func Panicf(format string, args ...interface{}) {
	defaultLogger.Panicf(format, args...)
}

// Panic logs to the default logger with a severity of "panic", and then
// panics with the supplied parameters. Logging only succeeds if the manager
// level is set to "panic" or lower.
// Arguments are handled in the same manner as fmt.Println.
func Panic(args ...interface{}) {
	defaultLogger.Panic(args...)
}

// Fatalf logs to the default logger with a severity of "fatal", and then
// calls Exit. Logging only succeeds if the manager level is set to "fatal"
// or lower.
// Arguments are handled in the same manner as fmt.Printf.
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

// Fatal logs to the default logger with a severity of "fatal", and then
// calls Exit. Logging only succeeds if the manager level is set to "fatal"
// or lower.
// Arguments are handled in the same manner as fmt.Println.
func Fatal(args ...interface{}) {
	defaultLogger.Fatal(args...)
}
