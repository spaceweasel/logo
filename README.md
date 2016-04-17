# Logo [![Build Status](https://travis-ci.org/spaceweasel/logo.svg?branch=master)](https://travis-ci.org/spaceweasel/logo) [![Coverage Status](http://codecov.io/github/spaceweasel/logo/coverage.svg?branch=master)](http://codecov.io/github/spaceweasel/logo?branch=master) [![GoDoc](http://img.shields.io/badge/godoc-reference-5272B4.svg)](https://godoc.org/github.com/spaceweasel/logo) [![MIT](https://img.shields.io/npm/l/express.svg)](https://github.com/spaceweasel/logo/blob/master/LICENSE)

Logo is a highly configurable logging package for Golang providing:

* Multiple named loggers
* Output formatting
* Shared appenders
 * Console
 * Rolling File (Buffered)
* Advanced severity level control
* Standard Golang log package hook


## The Global Logger

The global logger is a basic logger; it requires no setup and logs to the console by default.

```go
package main

import {
  "github.com/spaceweasel/logo"
}

func main() {
  logo.Debug("Starting application...")
}
```

The example logs the string `"Starting application..."` with the lowest level of severity, *debug*. This would produce output similar to:

`2016-01-09 05:04:05.000456 DEBUG (main.go:8) - Starting application...`

Other severities (in increasing severity) are: *info*, *warning*, *error*, *panic* and *fatal*. Panic and fatal have additional characteristics:

```go
package main

import {
  "github.com/spaceweasel/logo"
}

func main()

  // logs the message "application started"
  // then panics with the same message
  logo.Panic("application started")

  ...

  // logs the message "application started"
  // then exits the program
  logo.Fatal("application started")

}
```

There is a logging method for each of the severity levels. They have a variadic signature like `fmt.Println`, for example, `func Info(args ...interface{})`, enabling calls such as:

```go
logo.Debug("Starting calculation", currentUser, count)
...
logo.Warn("Excessive bandwidth used:", bw)  
```

There is an associated *formatted* method call for each severity which works in the same manner as `fmt.Printf`:

```go
logo.Debugf("Starting calculation (User: %s) - Input quantity: %d", currentUser, count)
...
logo.Warnf("Excessive bandwidth used: %f GB", bw)  
```

## Named Loggers

The Global Logger is ok for simple applications, but when your application has many areas it can be helpful to have more granularity over what messages are logged. For example, you might want to log debug messages from one part of your application, but only log warnings and errors from another. Named loggers are created with a minimum severity level; if the log method called is below the threshold, the request is ignored. Create as many loggers as your application requires, specifying the necessary severity level threshold.

```go
package main

import {
  "fmt"

  "github.com/spaceweasel/logo"
}

var (
  log           *logo.Logger
  validationLog *logo.Logger
  dbLog         *logo.Logger
)

func init(){
  // create a logger named "Main" with severity level of DEBUG
  log = logo.New("Main", "debug")

  // create a logger named "Validation" with severity level of INFO
  validationLog = logo.New("Validation", "info")

  // create a logger named "Database" with severity level of WARN
  dbLog = logo.New("Database", "warn")
}

func main() {
  // Ensures all data is written to disk.
  // Not strictly necessary when using default console appender,
  // but a good habit to get into if the appender is changed for a
  // file appender.
  defer logo.Close()

  log.Debug("Application started!")
  // rest of main...    
}

...

// in validation package...

func testString(s string)error{
  validationLog.Debugf("Testing string %q", s) // won't get logged - below threshold
  if len(s) == 0{
    validationLog.Error("Test string is empty")
    return fmt.Errorf("testme: empty string")       
  }
  return nil
}
```

## Master Severity Level

A *master* level can be set which overrides individual logger settings.

```go
// create a new named logger with minimum logging severity of INFO
log := logo.New("MyService", "info")

logo.SetManagerLevel("error") // sets master level
log.Debug("This message will never be logged") // below minimum level
log.Info("This message will not be logged") // below manager "master" level
log.Error("This message will still be logged though!")
```

__*Note that the master level affects the global logger too.*__

```go
// Global logger has default severity level of DEBUG  

logo.SetManagerLevel("error") // sets master level

logo.Debug("This message will never be logged") // below minimum level
logo.Info("This message will not be logged") // below manager "master" level
logo.Error("This message will still be logged though!")
```

## Appenders

### Appender Format

By default, new loggers will log to the console. They are initialized with the *ConsoleAppender* which uses the default format:

`"%date %severity (%file:%line) - %message%newline"`

This should be fairly self explanatory, but this means that each message will start on a new line and contain the date, severity, file and line location where the log request was made, together with actual message.

 (*Currently the format of the date is fixed as `yyyy-mm-dd hh:mm:ss.uuuuuu`, but this might become more configurable in the future*)

### Custom Formats

There are a few more format %-tags and most have single character shorthand equivalents:

Type | Tag | Shorthand | Description |Example output
---|---|---|---|---
Date | %date | %d | The log timestamp | 2016-01-09 05:04:05.000456
Severity | %severity | %s | Severity method used |INFO
Logger | %logger | | Name of the logger | MyService
File | %file | %f | Source filename where the log request was made| service.go
Line | %line | | Line in source file where log request was made| 345
Message | %message | %m | The concatenated log message details| The chickens have exploded
New Line | %newline | %n | Appends a \n|
Context | %context | %c | The logging context (see Logging Context) | CorrelationID: 45


The format of an appender can be changed using its `SetFormat` method:

```go
// in main
32  ...
33  // create calculator logger...
34  log := logo.New("Calculator", "info")
35  ConsoleAppender.SetFormat("%s %l (%f:%line): %m%n")
36  ...


// in divider.go
136 ...
137 func Divide(a, b int) (int, error){
138   if b == 0 {
139     log.Warn("Divide by zero")
140     return 0, fmt.Errorf("calculator: divide by zero")
141   }
142   return a/b, nil
143 }
144 ...

```
A divide by zero error will produce the log message:

`WARN Calculator (divider.go:139): Divide by zero`

### Additional Appenders

#### RollingFileAppender

Real applications normally need to persist their log data. `RollingFileAppender` is a buffered appender which writes to a named file; when a certain number of bytes have been written, it closes the file and creates a new file and writes to that instead. Rolling file appenders are created with a filename and a maximum file size (in MB). For example, `appender:= logo.RollingFileAppender("service.log", 5)`. This will use the name provided plus a time based suffix to create a new log file, when 5MB of data have been written, a new file is created as before, using the name provided and a new time based suffix. The name provided must include the full path and logfile name prefix; logging will be in the current directory if only filename is supplied. (*Note that currently old files are not deleted - you must do this manually*).

RollingFileAppender uses a large memory buffer to improve performance and reduce blocking. Data in the buffer is written to file every 30 seconds, or when a file is closed. Therefore, if you are tailing the log file, you won't necessarily see log messages immediately.

**IMPORTANT: Make sure logo.Close() is called before your application exits to ensure all data is written to disk!**

#### Assigning An Appender

Once an appender has been created, it must be added to the log manager before it can be assigned to a logger:

```go
// create new appender
appender := logo.RollingFileAppender("service.log", 5)

// add to the log manager with the alias "calc"
logo.AddAppender("calc", appender)

// create a new logger
log := logo.New("Calculator", "info")

// assign the "calc" appender to the logger
log.SetAppenders("calc")
```

Each logger is initialized with the console logger, but calling `SetAppenders` will overwrite this. The `SetAppenders` method is variadic, so you can include multiple appenders:

```go
a := logo.RollingFileAppender("service.log", 5)
logo.AddAppender("calc", a)
log := logo.New("Calculator", "info")

// "console" is the alias for the ConsoleAppender.
log.SetAppenders("calc", "console")

// This will log to the service.log file AND console.
// Note that the console will be updated immediately,
// but the file can take up to 30 seconds due to buffering
log.Error("Something bad happened")
```

We have already seen that the ConsoleAppender is used by all loggers by default. That is because appenders are not restricted to single loggers. In the same manner, a RollingFileAppender can be assigned to multiple loggers:

```go
var (
  mainLog       *logo.Logger
  validationLog *logo.Logger
  dbLog         *logo.Logger
)

func init(){
  mainLog = logo.New("Main", "debug")
  validationLog = logo.New("Validation", "info")
  dbLog = logo.New("Database", "warn")

  a := logo.RollingFileAppender("service.log", 5)
  logo.AddAppender("default", a)

  mainLog.SetAppenders("default", "console")
  validationLog.SetAppenders("default")
  dbLog.SetAppenders("default")      
}
```

You can even assign appenders to the global logger:

```go
a := logo.RollingFileAppender("service.log", 5)
logo.AddAppender("default", a)

// Notice calling package name "logo" not "log"
logo.SetAppenders("default", "console")  
```

### Filtering

Appenders can have filters which only permit messages to be written if their severity level matches that in the filter list. By default, no filtering occurs, and all messages passed to an appender are logged. To specify a filter, use the `SetFilters` method:

```go
ea := logo.RollingFileAppender("error.log", 5)
logo.AddAppender("errApp", ea)

// set a filter to ignore anything other than "ERROR" messages
ea.SetFilters("error")

log = logo.New("Main", "debug")
log.SetAppenders("errApp")

// This will not be logged - even though it is above the logger threshold
log.Info("This message will NOT be logged")

// This will be logged - filter only allows ERRORS
log.Error("This message WILL be logged")

// This will not be logged, however, it will still panic!
log.Panic("This message will NOT be logged though!")
```

Why might you want to set filtering on an appender? Flexibility.

Logo tries to provide as much flexibility as possible. Some system designers prefer to have a single log file to which everything is logged, regardless of the message severity. Others like to have different log files for different areas of their application, with each log file holding messages of varying severity. Finally, some designers prefer the approach of severity based log files. For example, `info.log`, `warning.log` and `error.log`; typically only errors are logged to `error.log`, but warnings *and* errors are logged to `warning.log` and everything is logged to `info.log`. This approach can be achieved using filters and the global logger:

```go
ea := logo.RollingFileAppender("error.log", 5)
logo.AddAppender("errors", ea)
ea.SetFilters("error", "panic", "fatal") // errors only

wa := logo.RollingFileAppender("warning.log", 5)
logo.AddAppender("warnings", wa)
wa.SetFilters("error", "warn", "panic", "fatal") // errors and warnings

ia := logo.RollingFileAppender("info.log", 5)
logo.AddAppender("information", ia)
// don't set any filters for the information appender - we want everything

// now set the global logger to use each appender
logo.SetAppenders("errors", "warnings", "information")

logo.Info("This will log to info.log only")
logo.Warn("This will log to info.log AND warning.log")
logo.Panic("This will log to all three log files, then panic!")
```

## Intercepting The Standard Golang Logger

Sometimes your application needs to log data from packages which use the standard "log" package, but are outside your control. You can use logo to intercept these log messages and have them sent to one or more appenders, by using the `CaptureStandardLog` method:

```go
a := logo.RollingFileAppender("service.log", 5)
logo.AddAppender("main", a)

// intercepts standard log package logging and writes to console and service.log
logo.CaptureStandardLog("main", "console")
```

## Context

In some of the earlier examples, log messages were created containing user information:

```go
log.Debug("Starting calculation", currentUser, count)
...
log.Debugf("Starting calculation (User: %s) - Input quantity: %d", currentUser, count)
```

It would be much cleaner if the log calls did not have to be cluttered with such contextual information. Logo provides the `WithContext` method which can be called on any named logger. The `WithContext` method returns a clone of the logger, but with embedded contextual information which can be included in appender message formats using the %context (or %c) tag.

```go
type context struct{
  userID string
}

// context objects can be anything but must implement the Stringer interface,
// as this is what will be used to write the data
func (c context) String() string {
  return fmt.Sprintf("User: %s", c.userID)
}


// create a new named logger
a := logo.RollingFileAppender("service.log", 5)
// include context in appender format (%c)
a.SetFormat("%l %s [%c] - %m%n")
logo.AddAppender("calc", a)
log := logo.New("Calculator", "info")
log.SetAppenders("calc")

...

ctx := context{4523}
clog := log.WithContext(ctx)

...

clog.Debug("Starting calculation...")
// Calculator DEBUG [User: 4523] - Starting calculation...

...

clog.Debug("Calculation  finished!")
// Calculator DEBUG [User: 4523] - Calculation  finished!
```
