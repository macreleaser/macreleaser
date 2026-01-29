package cli

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

// SetupLogger creates and configures a logger based on debug mode
func SetupLogger(debug bool) *logrus.Logger {
	logger := logrus.New()

	if debug {
		logger.SetLevel(logrus.DebugLevel)
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	} else {
		logger.SetLevel(logrus.InfoLevel)
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
		})
	}

	return logger
}

// ExitWithErrorf logs an error with the provided logger and exits with code 1
func ExitWithErrorf(logger *logrus.Logger, format string, args ...interface{}) {
	logger.Errorf(format, args...)
	os.Exit(1)
}

// ExitWithErrorNoLoggerf prints an error to stderr and exits with code 1
func ExitWithErrorNoLoggerf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}
