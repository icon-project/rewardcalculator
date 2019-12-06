package common

import (
	"log"
	"time"

	"github.com/natefinch/lumberjack"
)

type Log struct {
	logger lumberjack.Logger
}

const (
	timestampFormat = "2006-01-02 15:04:05.000"
	timestampMilliSecIndex = len(timestampFormat) - 4
)

func (l *Log) Write(p []byte) (n int, err error) {
	buf := make([]byte, 0)
	timestamp := time.Now().Format(timestampFormat) + " "
	buf = append(buf, timestamp...)
	buf = append(buf, p...)
	buf[timestampMilliSecIndex] = ','
	return l.logger.Write(buf)
}

func SetLog(file string, maxSize int, maxBackups int, localtime bool) {
	log.SetFlags(log.Lshortfile)
	rcLog := Log{
		lumberjack.Logger{
			Filename:   file,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			LocalTime:  localtime,
		},
	}
	log.SetOutput(&rcLog)
}

