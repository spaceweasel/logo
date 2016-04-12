package logo

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// IMPORTANT: Keep this as the first test to ensure line number remains the same
func TestLoggerOutputSetsMessageLine(t *testing.T) {
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "debug")
	l.SetAppenders("test")

	want := 19 // the next line
	l.Debug("A test message")

	messages := appender.logMessages

	if len(messages) != 1 {
		t.Errorf("Messages count got %d, want 1", len(messages))
		return
	}

	got := messages[0].line
	if got != want {
		t.Errorf("Message.line got %d, want %d", got, want)
	}
}

// IMPORTANT: Keep this as the second test to ensure line number remains the same
func TestDefaultLoggerOutputSetsMessageLine(t *testing.T) {
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	SetAppenders("test")

	want := 42 // the next line
	Debug("A test message")

	messages := appender.logMessages

	if len(messages) != 1 {
		t.Errorf("Messages count got %d, want 1", len(messages))
		return
	}

	got := messages[0].line
	if got != want {
		t.Errorf("Message.line got %d, want %d", got, want)
	}
}

func TestNewLoggerSetsLoggerName(t *testing.T) {
	want := "Test"
	defer reset()

	logger := New("Test", "test")
	got := logger.name

	if got != want {
		t.Errorf("Logger Name got %q, want %q", got, want)
	}
}

func TestNewLoggerSetsLoggerSeverityLevel(t *testing.T) {
	want := info
	defer reset()
	logger := New("Test", "INFO")
	got := logger.level

	if got != want {
		t.Errorf("Severity Level got %v, want %v", got, want)
	}
}

func TestNewLoggerSetsLoggerSeverityLevelRegardlessOfCase(t *testing.T) {
	want := info
	defer reset()

	logger := New("Test", "info")
	got := logger.level

	if got != want {
		t.Errorf("Severity Level got %v, want %v", got, want)
	}
}

func TestNewLoggerSetsLoggerSeverityLevelToDebugIfNotRecognised(t *testing.T) {
	want := debug
	defer reset()
	logger := New("Test", "unknown")
	got := logger.level

	if got != want {
		t.Errorf("Severity Level got %v, want %v", got, want)
	}
}

func TestInitialUseOfManagerReturnsLogManagerWithLevelDebug(t *testing.T) {
	want := debug
	got := manager.level

	if got != want {
		t.Errorf("Level got %v, want %v", got, want)
	}
}

func TestInitialUseOfManagerReturnsLogManagerWithInitializedAppendersMap(t *testing.T) {
	got := manager.appenders

	if got == nil {
		t.Errorf("Appenders got <nil>, want map[]")
	}
}

func TestInitialUseOfManagerReturnsLogManagerWithConsoleAppender(t *testing.T) {
	want := reflect.TypeOf(&consoleAppender{})
	appenders := manager.appenders
	a, _ := appenders["console"]
	got := reflect.TypeOf(a)

	if got != want {
		t.Errorf("ConsoleAppender got %v, want %v", got, want)
	}
}

func TestInitialUseOfDefaultLoggerHasSeverityLevelDebug(t *testing.T) {
	want := debug
	dl := defaultLogger

	if dl == nil {
		t.Errorf("DefaultLogger is <nil>")
		return
	}

	got := dl.level
	if got != want {
		t.Errorf("Level got %v, want %v", got, want)
	}
}

func TestDefaultLoggerHasEmptyName(t *testing.T) {
	want := ""
	dl := defaultLogger

	if dl == nil {
		t.Errorf("DefaultLogger is <nil>")
		return
	}

	got := dl.name
	if got != want {
		t.Errorf("DefaultLogger name got %q, want %q", got, want)
	}
}

func TestAddAppenderAddsToCollection(t *testing.T) {
	want := reflect.TypeOf(&emptyAppender{})
	defer reset()
	AddAppender("Test", EmptyAppender)
	appenders := manager.appenders
	if len(appenders) != 2 {
		t.Errorf("Appenders count got %d, want 2", len(appenders))
		return
	}

	a, _ := appenders["Test"]
	got := reflect.TypeOf(a)

	if got != want {
		t.Errorf("Test appender got %v, want %v", got, want)
	}
}

func TestAddAppenderReturnsErrorWhenCalledWithSameName(t *testing.T) {
	want := "appender already exist"
	defer reset()

	err := AddAppender("console", EmptyAppender)
	if err == nil {
		t.Errorf("AddAppender error got <nil>, want %q", want)
	}
	got := err.Error()

	if got != want {
		t.Errorf("AddAppender error got %q, want %q", got, want)
	}
}

func TestAddAppenderUpdatesAppenderWithSameName(t *testing.T) {
	want := ConsoleAppender
	defer reset()
	//conApp := ConsoleAppender
	AddAppender("console", EmptyAppender)

	appenders := manager.appenders
	if len(appenders) != 1 {
		t.Errorf("Appenders count got %d, want 1", len(appenders))
		return
	}

	a, _ := appenders["console"]
	got := a

	if got != want {
		t.Errorf("Console appender got %v, want %v", got, want)
	}
}

func TestNewLoggerAddsConsoleAppender(t *testing.T) {
	want := reflect.TypeOf(&consoleAppender{})
	defer reset()
	l := New("Test", "debug")
	appenders := l.a
	if len(appenders) != 1 {
		t.Errorf("Appenders count got %d, want 1", len(appenders))
		return
	}
	got := reflect.TypeOf(appenders[0])

	if got != want {
		t.Errorf("Appender got %v, want %v", got, want)
	}
}

func TestLoggerSetAppendersOverwritesExisting(t *testing.T) {
	want := reflect.TypeOf(&emptyAppender{})
	defer reset()
	AddAppender("TestAppender", EmptyAppender)
	l := New("Test", "debug")
	l.SetAppenders("TestAppender")

	appenders := l.a
	if len(appenders) != 1 {
		t.Errorf("Appenders count got %d, want 1", len(appenders))
		return
	}
	got := reflect.TypeOf(appenders[0])

	if got != want {
		t.Errorf("Appender got %v, want %v", got, want)
	}
}

func TestLoggerSetAppendersAcceptsMultipleAppenders(t *testing.T) {
	want := "*logo.emptyAppender-*logo.consoleAppender"
	defer reset()
	AddAppender("TestAppender", EmptyAppender)
	l := New("Test", "debug")
	l.SetAppenders("TestAppender", "console")

	appenders := l.a
	if len(appenders) != 2 {
		t.Errorf("Appenders count got %d, want 1", len(appenders))
		return
	}
	n1 := reflect.TypeOf(appenders[0]).String()
	n2 := reflect.TypeOf(appenders[1]).String()
	got := n1 + "-" + n2

	if got != want {
		t.Errorf("Appenders got %v, want %v", got, want)
	}
}

func TestLoggerSetAppendersPanicsWhenUnrecognisedAppender(t *testing.T) {
	want := `unrecognised appender, [UnknownAppender]`
	defer func() {
		if r := recover(); r != nil {
			got := r
			if got != want {
				t.Errorf("Error got %q, want %q", got, want)
				return
			}
		} else {
			t.Errorf("The code did not panic")
		}
	}()
	defer reset()

	l := New("Test", "debug")
	l.SetAppenders("UnknownAppender")
}

func TestLogManagerCloseCallsAllAppenders(t *testing.T) {
	defer reset()
	appenders := []Appender{newTestAppender(), newTestAppender(), newTestAppender(), newTestAppender()}
	names := []string{}
	for i, a := range appenders {
		n := fmt.Sprintf("testAppender%d", i)
		AddAppender(n, a)
		names = append(names, n)
	}
	Close()

	want := true
	for i, a := range appenders {
		got := (a.(*testAppender)).Closed

		if got != want {
			t.Errorf("%s.Close() called: got %t, want %t", names[i], got, want)
		}
	}
}

func TestLoggerDebugfSendsPopulatedMsgToAppender(t *testing.T) {
	defer reset()
	timenow = func() time.Time {
		t, _ := time.Parse("2006-01-02T15:04:05.999999999", "2016-11-19T15:14:15.123456789")
		return t
	}

	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "debug")
	l.SetAppenders("test")
	l.Debugf("A test message %d", 56)

	messages := appender.logMessages

	if len(messages) != 1 {
		t.Errorf("Messages count: got %d, want 1", len(messages))
		return
	}

	var tests = []struct {
		property string
		f        func(*LogMessage) interface{}
		want     interface{}
	}{
		{"format", func(m *LogMessage) interface{} { return m.format }, "A test message %d"},
		{"args.Count()", func(m *LogMessage) interface{} { return len(m.args) }, 1},
		{"args[0]", func(m *LogMessage) interface{} {
			if len(m.args) > 0 {
				return m.args[0]
			}
			return 0
		}, 56},
		{"severity", func(m *LogMessage) interface{} { return m.severity }, "DEBUG"},
		{"name", func(m *LogMessage) interface{} { return m.name }, "Test"},
		{"file", func(m *LogMessage) interface{} { return m.file }, "log_test.go"},
		{"ctx", func(m *LogMessage) interface{} { return m.ctx }, ""},
		{"timestamp", func(m *LogMessage) interface{} { return string(m.timestamp) }, "2016-11-19 15:14:15.123456"},
	}

	for _, test := range tests {
		if got := test.f(messages[0]); got != test.want {
			t.Errorf("Message.%s: got %v, want %v", test.property, got, test.want)
		}
	}
}

func TestDefaultLoggerDebugfSendsPopulatedMsgToAppender(t *testing.T) {
	defer reset()
	timenow = func() time.Time {
		t, _ := time.Parse("2006-01-02T15:04:05.999999999", "2016-11-19T15:14:15.123456789")
		return t
	}

	appender := newTestAppender()
	AddAppender("test", appender)
	SetAppenders("test")
	Debugf("A test message %d", 56)

	messages := appender.logMessages

	if len(messages) != 1 {
		t.Errorf("Messages count: got %d, want 1", len(messages))
		return
	}

	var tests = []struct {
		property string
		f        func(*LogMessage) interface{}
		want     interface{}
	}{
		{"format", func(m *LogMessage) interface{} { return m.format }, "A test message %d"},
		{"args.Count()", func(m *LogMessage) interface{} { return len(m.args) }, 1},
		{"args[0]", func(m *LogMessage) interface{} {
			if len(m.args) > 0 {
				return m.args[0]
			}
			return 0
		}, 56},
		{"severity", func(m *LogMessage) interface{} { return m.severity }, "DEBUG"},
		{"name", func(m *LogMessage) interface{} { return m.name }, ""},
		{"file", func(m *LogMessage) interface{} { return m.file }, "log_test.go"},
		{"ctx", func(m *LogMessage) interface{} { return m.ctx }, ""},
		{"timestamp", func(m *LogMessage) interface{} { return string(m.timestamp) }, "2016-11-19 15:14:15.123456"},
	}

	for _, test := range tests {
		if got := test.f(messages[0]); got != test.want {
			t.Errorf("Message.%s: got %v, want %v", test.property, got, test.want)
		}
	}
}

func TestLoggerSendsMsgToEachAppender(t *testing.T) {
	defer reset()
	appenders := []Appender{newTestAppender(), newTestAppender(), newTestAppender(), newTestAppender()}
	names := []string{}
	for i, a := range appenders {
		n := fmt.Sprintf("testAppender%d", i)
		AddAppender(n, a)
		names = append(names, n)
	}

	l := New("Test", "debug")
	l.SetAppenders(names...)
	l.Debugf("A test message %d", 56)

	var tests = []struct {
		property string
		f        func(*LogMessage) interface{}
		want     interface{}
	}{
		{"format", func(m *LogMessage) interface{} { return m.format }, "A test message %d"},
		{"args.Count()", func(m *LogMessage) interface{} { return len(m.args) }, 1},
		{"args[0]", func(m *LogMessage) interface{} {
			if len(m.args) > 0 {
				return m.args[0]
			}
			return 0
		}, 56},
		{"severity", func(m *LogMessage) interface{} { return m.severity }, "DEBUG"},
		{"name", func(m *LogMessage) interface{} { return m.name }, "Test"},
		{"file", func(m *LogMessage) interface{} { return m.file }, "log_test.go"},
	}

	for i, a := range appenders {
		messages := (a.(*testAppender)).logMessages

		if len(messages) != 1 {
			t.Errorf("%s messages count got %d, want 1", names[i], len(messages))
			return
		}
		for _, test := range tests {
			if got := test.f(messages[0]); got != test.want {
				t.Errorf("Message.%s got %v, want %v", test.property, got, test.want)
			}
		}
	}
}

func TestLoggerOutputPadsDateElementsWithLeadingZeros(t *testing.T) {
	want := "2016-01-09 05:04:05.000456"

	timenow = func() time.Time {
		t, _ := time.Parse("2006-01-02T15:04:05.999999999", "2016-01-09T05:04:05.000456789")
		return t
	}
	defer reset()

	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "debug")
	l.SetAppenders("test")

	l.Debug("A test message")

	messages := appender.logMessages

	if len(messages) != 1 {
		t.Errorf("Messages count: got %d, want 1", len(messages))
		return
	}

	got := string(messages[0].timestamp)
	if got != want {
		t.Errorf("Message.timestamp: got %q, want %q", got, want)
	}
}

type logContext struct {
	correlationID int
}

func (l logContext) String() string {
	return fmt.Sprintf("CorrelationId: %d", l.correlationID)
}

func TestLoggerWithContextSetsLogMessageContext(t *testing.T) {
	want := "CorrelationId: 45"
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "debug")
	l.SetAppenders("test")

	cl := l.WithContext(logContext{correlationID: 45})
	cl.Debugf("A test message %d", 56)

	messages := appender.logMessages

	if len(messages) != 1 {
		t.Errorf("Messages count: got %d, want 1", len(messages))
		return
	}

	got := messages[0].ctx
	if got != want {
		t.Errorf("Message.ctx: got %q, want %q", got, want)
	}
}

func TestLoggerWithContextUsesParentAppenders(t *testing.T) {
	want := "A test message %dAnother test message %d"
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "debug")
	l.SetAppenders("test")

	cl := l.WithContext(logContext{correlationID: 45})
	cl.Debugf("A test message %d", 56)
	l.Debugf("Another test message %d", 23)

	messages := appender.logMessages

	if len(messages) != 2 {
		t.Errorf("Messages count: got %d, want 2", len(messages))
		return
	}

	got := messages[0].format + messages[1].format
	if got != want {
		t.Errorf("Message formats: got %q, want %q", got, want)
	}
}

func TestLoggerDebugfIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "info")
	l.SetAppenders("test")

	l.Debugf("A test message %d", 56)

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerWithContextIgnoresIfParentLoggerLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "info")
	l.SetAppenders("test")

	cl := l.WithContext(logContext{correlationID: 45})
	cl.Debugf("A test message %d", 56)

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerDebugIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "info")
	l.SetAppenders("test")

	l.Debug("A test message")

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerInfofIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "fatal")
	l.SetAppenders("test")

	l.Infof("A test message %d", 56)

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerInfoIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "fatal")
	l.SetAppenders("test")

	l.Info("A test message")

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerWarnfIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "fatal")
	l.SetAppenders("test")

	l.Warnf("A test message %d", 56)

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerWarnIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "fatal")
	l.SetAppenders("test")

	l.Warn("A test message")

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerErrorfIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "fatal")
	l.SetAppenders("test")

	l.Errorf("A test message %d", 56)

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerErrorIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "fatal")
	l.SetAppenders("test")

	l.Error("A test message")

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerPanicfIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "fatal")
	l.SetAppenders("test")

	func() {
		defer func() { recover() }()
		l.Panicf("A test message %d", 56)
	}()

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerPanicIgnoresWhenLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "fatal")
	l.SetAppenders("test")

	func() {
		defer func() { recover() }()
		l.Panic("A test message")
	}()

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerIgnoresWhenManagerLevelSetAbove(t *testing.T) {
	want := 0
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "debug")
	l.SetAppenders("test")

	SetManagerLevel("fatal")

	l.Info("A test message")

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerDoesNotIgnoreWhenManagerLevelSetEqual(t *testing.T) {
	want := 1
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "debug")
	l.SetAppenders("test")

	SetManagerLevel("info")

	l.Info("A test message")

	got := len(appender.logMessages)

	if got != want {
		t.Errorf("Messages count: got %d, want %d", got, want)
	}
}

func TestLoggerSetsLogMessageSeverity(t *testing.T) {
	exit = func(i int) {}
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "debug")
	l.SetAppenders("test")
	s := "test"

	var tests = []struct {
		property string
		f        func()
		want     string
	}{
		{"Debug", func() { l.Debug("") }, "DEBUG"},
		{"Debugf", func() { l.Debugf(s, 8) }, "DEBUG"},
		{"Info", func() { l.Info("") }, "INFO"},
		{"infof", func() { l.Infof(s, 8) }, "INFO"},
		{"Warn", func() { l.Warn("") }, "WARN"},
		{"Warnf", func() { l.Warnf(s, 8) }, "WARN"},
		{"Error", func() { l.Error("") }, "ERROR"},
		{"Errorf", func() { l.Errorf(s, 8) }, "ERROR"},
		{"Panic", func() { l.Panic("") }, "PANIC"},
		{"Panicf", func() { l.Panicf(s, 8) }, "PANIC"},
		{"Fatal", func() { l.Fatal("") }, "FATAL"},
		{"Fatalf", func() { l.Fatalf(s, 8) }, "FATAL"},
	}

	for i, test := range tests {
		// run test in isolation for log.Panic and log.Panicf
		func() {
			defer func() { recover() }()
			test.f()
		}()

		if got := appender.logMessages[i].severity; got != test.want {
			t.Errorf("%s: got %v, want %v", test.property, got, test.want)
		}
	}
}

func TestLoggerFatalExitsWithCorrectCode(t *testing.T) {
	var ec int
	exit = func(i int) { ec = i }
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "debug")
	l.SetAppenders("test")
	s := "test"

	var tests = []struct {
		property string
		f        func()
		want     int
	}{
		{"Fatal", func() {
			ec = 0
			l.Fatal("")
		}, 1},
		{"Fatalf", func() {
			ec = 0
			l.Fatalf(s, 8)
		}, 1},
	}

	for _, test := range tests {
		test.f()

		if got := ec; got != test.want {
			t.Errorf("%s: got %v, want %v", test.property, got, test.want)
		}
	}
}

func TestLoggerPanicCallsStillPanicWhenSeverityAbovePanic(t *testing.T) {
	exit = func(i int) {}
	defer reset()
	appender := newTestAppender()
	AddAppender("test", appender)
	l := New("Test", "fatal")
	l.SetAppenders("test")

	var tests = []struct {
		property string
		f        func()
		want     string
	}{
		{"Panic", func() { l.Panic("chickens have escaped") }, "chickens have escaped"},
		{"Panicf", func() { l.Panicf("invalid %s type, %s", "chicken", "fish") }, "invalid chicken type, fish"},
	}

	for _, test := range tests {
		func() {
			defer func() {
				if r := recover(); r != nil {
					got := r
					if got != test.want {
						t.Errorf("%s: got %v, want %v", test.property, got, test.want)
					}
				} else {
					t.Errorf("%s did not panic", test.property)
				}
			}()
			test.f()
		}()

	}
}

func TestDefaultLoggerSetsLogMessageSeverity(t *testing.T) {
	defer reset()
	exit = func(i int) {}

	appender := newTestAppender()
	AddAppender("test", appender)
	SetAppenders("test")

	s := "test"

	var tests = []struct {
		property string
		f        func()
		want     string
	}{
		{"Debug", func() { Debug("") }, "DEBUG"},
		{"Debugf", func() { Debugf(s, 8) }, "DEBUG"},
		{"Info", func() { Info("") }, "INFO"},
		{"infof", func() { Infof(s, 8) }, "INFO"},
		{"Warn", func() { Warn("") }, "WARN"},
		{"Warnf", func() { Warnf(s, 8) }, "WARN"},
		{"Error", func() { Error("") }, "ERROR"},
		{"Errorf", func() { Errorf(s, 8) }, "ERROR"},
		{"Panic", func() { Panic("") }, "PANIC"},
		{"Panicf", func() { Panicf(s, 8) }, "PANIC"},
		{"Fatal", func() { Fatal("") }, "FATAL"},
		{"Fatalf", func() { Fatalf(s, 8) }, "FATAL"},
	}

	for i, test := range tests {
		// run test in isolation for log.Panic and log.Panicf
		func() {
			defer func() { recover() }()
			test.f()
		}()
		if got := appender.logMessages[i].severity; got != test.want {
			t.Errorf("%s: got %v, want %v", test.property, got, test.want)
		}
	}
}

func reset() {
	manager = newLogManager()
	defaultLogger = newDefaultLogger()
	timenow = time.Now
}
