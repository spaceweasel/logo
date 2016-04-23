package logo

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
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
	none
)

var severityName = []string{
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
	"PANIC",
	"FATAL",
	"",
}

func severityFromName(n string) severity {
	n = strings.ToUpper(n)
	for i, s := range severityName {
		if s == n {
			return severity(i)
		}
	}
	// TODO: Should this return an error instead?
	return none
}

type logManager struct {
	appenders map[string]Appender
	level     severity
	loggers   map[string]*Logger
}

var manager = newLogManager()
var defaultLogger = newDefaultLogger()

func newLogManager() *logManager {
	m := logManager{
		appenders: make(map[string]Appender),
		loggers:   make(map[string]*Logger),
	}
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
	timestamp time.Time
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
		m = &LogMessage{}
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

// New returns a new logger instance.
// Name can be any (non empty) string and is included in any log output
// (if specified in the appender format string). Normally this would be
// the name of a package using the logger. Level specifies the minimum
// severity for successful logging.
// For example, if the Warn() or Info() methods are called on a logger
// with severity level "info", then logging will be successful, but calls
// to Debug() will not.
// New panics if a logger with the same name has been created previously.
func New(name string, level string) *Logger {
	if _, ok := manager.loggers[name]; ok {
		panic("duplicate logger name")
	}
	sev := severityFromName(level)
	logger := &Logger{
		level:     sev,
		name:      name,
		appenders: []Appender{ConsoleAppender},
		callDepth: 2,
	}
	manager.loggers[name] = logger
	return logger
}

// LoggerByName returns a pointer to a logger named n.
// LoggerByName returns false and a nil pointer if no
// such named logger exists.
func LoggerByName(n string) (*Logger, bool) {
	l, err := manager.loggers[n]
	return l, err
}

// Logger is a named logger owned by the log manager. The log manager has
// at least one default logger instance, but additional named loggers can
// be created with the New() method.
type Logger struct {
	level     severity
	name      string
	appenders []Appender
	context   string
	callDepth int
}

func fileline(depth int) (string, int) {
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		file = "???"
		line = 0
	} else {
		slash := strings.LastIndex(file, "/")
		if slash >= 0 {
			file = file[slash+1:]
		}
	}
	return file, line
}

func (l *Logger) output(file string, line int, s severity, format string, args ...interface{}) {
	if manager.level > debug && s < manager.level {
		return
	}
	msg := getMessage()
	msg.severity = s
	msg.name = l.name
	msg.ctx = l.context
	msg.args = args
	msg.format = format
	msg.file = file
	msg.line = line
	msg.timestamp = timenow()

	for _, a := range l.appenders {
		a.Append(msg)
	}
	putMessage(msg)
}

// SetAppenders specifies one or more appenders that the logger should use.
// Appenders are specified by their string name, and must have been added to
// the log manager previously using the AddAppender method.
// SetAppenders will panic if an appender name is not recognised.
func (l *Logger) SetAppenders(names ...string) error {
	l.appenders = []Appender{}
	var ok bool
	for _, n := range names {
		var a Appender
		if a, ok = manager.appenders[n]; !ok {
			return fmt.Errorf("unrecognised appender, [%s]", n)
		}
		l.appenders = append(l.appenders, a)
	}
	return nil
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
		level:     l.level,
		name:      l.name,
		appenders: l.appenders,
		context:   context.String(),
	}
}

// Debug logs with a severity of "debug". Logging only succeeds if both
// the logger level (and manager level) are set to "debug".
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Debug(args ...interface{}) {
	if l.level > debug {
		return
	}
	file, line := fileline(l.callDepth)
	l.output(file, line, debug, "", args...)
}

// Debugf logs with a severity of "debug". Logging only succeeds if both
// the logger level (and manager level) are set to "debug".
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.level > debug {
		return
	}
	file, line := fileline(l.callDepth)
	l.output(file, line, debug, format, args...)
}

// Info logs with a severity of "info". Logging only succeeds if both the
// logger level (and manager level) are set to "info" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Info(args ...interface{}) {
	if l.level > info {
		return
	}
	file, line := fileline(l.callDepth)
	l.output(file, line, info, "", args...)
}

// Infof logs with a severity of "info". Logging only succeeds if both the
// logger level (and manager level) are set to "info" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Infof(format string, args ...interface{}) {
	if l.level > info {
		return
	}
	file, line := fileline(l.callDepth)
	l.output(file, line, info, format, args...)
}

// Warn logs with a severity of "warn". Logging only succeeds if both the
// logger level (and manager level) are set to "warn" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Warn(args ...interface{}) {
	if l.level > warn {
		return
	}
	file, line := fileline(l.callDepth)
	l.output(file, line, warn, "", args...)
}

// Warnf logs with a severity of "warn". Logging only succeeds if both the
// logger level (and manager level) are set to "warn" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Warnf(format string, args ...interface{}) {
	if l.level > warn {
		return
	}
	file, line := fileline(l.callDepth)
	l.output(file, line, warn, format, args...)
}

// Error logs with a severity of "error". Logging only succeeds if both the
// logger level (and manager level) are set to "error" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Error(args ...interface{}) {
	if l.level > errorMsg {
		return
	}
	file, line := fileline(l.callDepth)
	l.output(file, line, errorMsg, "", args...)
}

// Errorf logs with a severity of "error". Logging only succeeds if both the
// logger level (and manager level) are set to "error" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Errorf(format string, args ...interface{}) {
	if l.level > errorMsg {
		return
	}
	file, line := fileline(l.callDepth)
	l.output(file, line, errorMsg, format, args...)
}

// Panic logs with a severity of "panic", and then panics with the
// supplied parameters. Logging only succeeds if both the
// logger level (and manager level) are set to "panic" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Panic(args ...interface{}) {
	if l.level <= panicMsg {
		file, line := fileline(l.callDepth)
		l.output(file, line, panicMsg, "", args...)
	}
	panic(fmt.Sprint(args...))
}

// Panicf logs with a severity of "panic", and then panics with the
// supplied parameters. Logging only succeeds if both the
// logger level (and manager level) are set to "panic" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Panicf(format string, args ...interface{}) {
	if l.level <= panicMsg {
		file, line := fileline(l.callDepth)
		l.output(file, line, panicMsg, format, args...)
	}
	panic(fmt.Sprintf(format, args...))
}

var exit = func(i int) { os.Exit(i) } // to facilitate testing

// Fatal logs with a severity of "fatal", and then calls Exit.
// Logging only succeeds if both the logger level (and manager level)
// are set to "fatal" or lower.
// Arguments are handled in the same manner as fmt.Println.
func (l *Logger) Fatal(args ...interface{}) {
	file, line := fileline(l.callDepth)
	l.output(file, line, fatal, "", args...)
	Close()
	exit(1)
}

// Fatalf logs with a severity of "fatal", and then calls Exit.
// Logging only succeeds if both the logger level (and manager level)
// are set to "fatal" or lower.
// Arguments are handled in the same manner as fmt.Printf.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	file, line := fileline(l.callDepth)
	l.output(file, line, fatal, format, args...)
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

// CaptureStandardLog hooks into the standard go log package and redirects
// the output to appenders.
func CaptureStandardLog(appenders ...string) {
	b := bridge{
		Logger: Logger{level: none},
	}
	b.SetAppenders(appenders...)

	log.SetFlags(log.Lshortfile)
	log.SetOutput(b)
}

type bridge struct {
	Logger
}

func (l bridge) Write(b []byte) (n int, err error) {
	var msg string
	file := "???"
	line := 0

	// format is "file.go:1234: message"
	parts := bytes.SplitN(b, []byte{':'}, 3)
	if len(parts) != 3 || len(parts[0]) == 0 || len(parts[2]) == 0 {
		msg = fmt.Sprintf("(Invalid log format): %s", b)
	} else {
		file = string(parts[0])
		msg = string(parts[2])
		line, err = strconv.Atoi(string(parts[1]))
		if err != nil {
			msg = fmt.Sprintf("(Invalid line number): %s", b)
		}
	}
	msg = strings.TrimSpace(msg)
	l.output(file, line, l.level, msg)

	return len(b), nil
}
