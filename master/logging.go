package master

import (
	"fmt"
	"os"
	"regexp"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var levelRegex *regexp.Regexp

func init() {
	var err error
	levelRegex, err = regexp.Compile("level=([a-z]+)")
	if err != nil {
		logrus.WithError(err).Fatal("Cannot setup log level")
	}
}

// Writes error log entries to error output and everything else to standart output
// inspired by https://huynvk.dev/blog/4-tips-for-logging-on-gcp-using-golang-and-logrus
type LogWriter struct {
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	level := logrus.FatalLevel
	matches := levelRegex.FindStringSubmatch(string(p))
	if len(matches) > 1 {
		level, _ = logrus.ParseLevel(matches[1])
	}

	switch level {
	case logrus.ErrorLevel, logrus.WarnLevel, logrus.FatalLevel, logrus.PanicLevel:
		return os.Stderr.Write(p)
	default:
		return os.Stdout.Write(p)
	}
}

type StacktraceHook struct {
}

func (h *StacktraceHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *StacktraceHook) Fire(e *logrus.Entry) error {
	if v, found := e.Data[logrus.ErrorKey]; found {
		if err, iserr := v.(error); iserr {
			var tracer interface {
				StackTrace() errors.StackTrace
			}
			if !errors.As(err, &tracer) {
				return nil
			} else {
				stack := fmt.Sprintf("%+v", tracer.StackTrace())
				e.Data["stacktrace"] = stack
			}
		}
	}
	return nil
}
