package sesame

import (
	"fmt"
	"os"
)

// LogLevel is a level of the logging.
type LogLevel int

const (
	// LogLevelDebug is a debug level log.
	LogLevelDebug LogLevel = -4

	// LogLevelInfo is an info level log.
	LogLevelInfo LogLevel = 0

	// LogLevelWarn is a warning level log.
	LogLevelWarn LogLevel = 4

	// LogLevelError is an error level log.
	LogLevelError LogLevel = 8
)

// Log is a function for the logging.
type Log func(level LogLevel, format string, args ...any)

// StdLog is a [Log] that writes to stdout and stderr.
func StdLog(level LogLevel, format string, args ...any) {
	out := os.Stdout
	if level >= LogLevelWarn {
		out = os.Stderr
	}
	if level >= LogEnabledFor {
		fmt.Fprintf(out, format, args...)
		fmt.Fprint(out, "\n")
	}
}

// LogFunc is a [Log] used in this package.
var LogFunc Log = StdLog

// LogEnabledFor is a threshold for the logging.
var LogEnabledFor = LogLevelInfo
