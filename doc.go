// Package logo is a global log manager, providing multiple named logger
// instances in addition to a default global logger. Each logger has an
// associated severity level, enabling lower level messages to be turned
// on for certain areas of your application. In addition, the log manager
// has its own master level which overrides that of individual loggers.
// Loggers write to one or more appenders, each of which can be configured
// to use a certain message format, or even filter messages; accepting only
// messages of a specific severity.
package logo
