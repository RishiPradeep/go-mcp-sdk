package logger

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

type GinStyleFormatter struct{}

func (f *GinStyleFormatter) Format(entry *log.Entry) ([]byte, error) {
	levelColor := "\033[37m" // Default white
	resetColor := "\033[0m"

	switch entry.Level {
	case log.InfoLevel:
		levelColor = "\033[32m" // Green
	case log.WarnLevel:
		levelColor = "\033[33m" // Yellow
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		levelColor = "\033[31m" // Red
	case log.DebugLevel:
		levelColor = "\033[36m" // Cyan
	}

	// Format timestamp similar to Gin logs
	timestamp := entry.Time.Format("2006/01/02 - 15:04:05")

	// Uppercase and pad log level for alignment
	level := fmt.Sprintf("% -5s", strings.ToUpper(entry.Level.String()))

	// Format final message
	return []byte(fmt.Sprintf("%s[%s] %s%s | %s\n",
		levelColor, timestamp, level, resetColor, entry.Message)), nil
}

func init() {

	log.SetOutput(os.Stdout)
	log.SetFormatter(&GinStyleFormatter{})
	log.SetReportCaller(false) // Remove file:line
	log.SetLevel(log.InfoLevel)
}