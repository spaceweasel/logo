package logo

import "testing"

func testMessage() *LogMessage {
	d := []byte("2016-04-09 18:03:28.342017")
	msg := LogMessage{
		severity:  info,
		name:      "Logger",
		ctx:       "{ctx: 2}",
		args:      []interface{}{34, 56},
		format:    "Test %d (%d)",
		timestamp: d,
		file:      "sample.go",
		line:      456,
	}
	return &msg
}

func TestFormatterNames(t *testing.T) {
	var tests = []struct {
		property string
		f        Formatter
		want     string
	}{
		{"literalFormatter", &literalFormatter{}, ""},
		{"dateFormatter", &dateFormatter{}, "date,d,"},
		{"severityFormatter", &severityFormatter{}, "severity,s,"},
		{"loggerFormatter", &loggerFormatter{}, "logger,"},
		{"fileFormatter", &fileFormatter{}, "file,f,"},
		{"lineFormatter", &lineFormatter{}, "line,"},
		{"contextFormatter", &contextFormatter{}, "context,c,"},
		{"messageFormatter", &messageFormatter{}, "message,m,"},
		{"newlineFormatter", &newlineFormatter{}, "newline,n,"},
	}

	for _, test := range tests {
		got := ""
		for _, n := range test.f.Names() {
			got += n + ","
		}
		if got != test.want {
			t.Errorf("%s Names() got %q, want %q", test.property, got, test.want)
		}
	}
}

func TestFormatterResults(t *testing.T) {
	var tests = []struct {
		property string
		f        Formatter
		want     string
	}{
		{"literalFormatter", &literalFormatter{s: " Test:["}, " Test:["},
		{"dateFormatter", &dateFormatter{}, "2016-04-09 18:03:28.342017"},
		{"severityFormatter", &severityFormatter{}, "INFO"},
		{"loggerFormatter", &loggerFormatter{}, "Logger"},
		{"fileFormatter", &fileFormatter{}, "sample.go"},
		{"lineFormatter", &lineFormatter{}, "456"},
		{"contextFormatter", &contextFormatter{}, "{ctx: 2}"},
		{"messageFormatter", &messageFormatter{}, "Test 34 (56)"},
		{"newlineFormatter", &newlineFormatter{}, "\n"},
	}

	for _, test := range tests {
		m := testMessage()
		test.f.Format(m)
		got := string(m.Bytes())
		if got != test.want {
			t.Errorf("%s got %q, want %q", test.property, got, test.want)
		}
	}
}

func TestMessageFormatterResults(t *testing.T) {
	f := &messageFormatter{}

	var tests = []struct {
		format string
		args   []interface{}
		want   string
	}{
		{"Test-%d (%s): %d", []interface{}{45, "test", 0}, "Test-45 (test): 0"},
		{"", []interface{}{45, "test", 98, "G", true}, "45test98Gtrue"},
		{"Test my chickens", []interface{}{}, "Test my chickens"},
		{"Test my chickens", nil, "Test my chickens"},
	}

	for _, test := range tests {
		m := testMessage()
		m.format = test.format
		m.args = test.args
		f.Format(m)
		got := string(m.Bytes())
		if got != test.want {
			t.Errorf("Format got %q, want %q", got, test.want)
		}
	}
}

func TestExtractor(t *testing.T) {
	var tests = []struct {
		format string
		want   string
	}{
		{"blah blah more blah", "blah blah more blah"},
		{"blah %d more", "blah 2016-04-09 18:03:28.342017 more"},
		{"blah %date more", "blah 2016-04-09 18:03:28.342017 more"},
		{"blah %severity more", "blah INFO more"},
		{"blah %s more", "blah INFO more"},
		{"blah %logger more", "blah Logger more"},
		{"blah %file more", "blah sample.go more"},
		{"blah %f more", "blah sample.go more"},
		{"blah %line more", "blah 456 more"},
		{"blah %context more", "blah {ctx: 2} more"},
		{"blah %c more", "blah {ctx: 2} more"},
		{"blah %message more", "blah Test 34 (56) more"},
		{"blah %m more", "blah Test 34 (56) more"},
		{"blah %newline more", "blah \n more"},
		{"blah %n more", "blah \n more"},
		{"blah %n more%n", "blah \n more\n"},
		{"%d %s %logger (%f:%line)%n", "2016-04-09 18:03:28.342017 INFO Logger (sample.go:456)\n"},
		{"%s%s %%logger (%f:%line)%n", "INFOINFO %logger (sample.go:456)\n"},
		{"blah more%", "blah more%"},
		{"blah more%%", "blah more%"},
	}

	for _, test := range tests {
		m := testMessage()
		formatters, _ := extract(test.format)
		for _, f := range formatters {
			f.Format(m)
		}
		got := string(m.Bytes())
		if got != test.want {
			t.Errorf("%s got %q, want %q", test.format, got, test.want)
		}
	}
}

func TestExtractorReturnsErrorWhenInvalidSyntax(t *testing.T) {
	var tests = []struct {
		format string
		want   string
	}{
		{"bla%h blah", "invalid syntax at position 3, bla%h blah"},
		{"bla%%%h blah", "invalid syntax at position 5, bla%%%h blah"},
		{"%blah blah", "invalid syntax at position 0, %blah blah"},
	}

	for _, test := range tests {
		_, err := extract(test.format)
		if err == nil {
			t.Errorf("%s got <nil>, want %q", test.format, test.want)
		}
		got := err.Error()
		if got != test.want {
			t.Errorf("%s got %q, want %q", test.format, got, test.want)
		}
	}
}
