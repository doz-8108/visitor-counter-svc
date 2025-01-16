package utils

import (
	"fmt"
	"os"
	"path"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func SetUpLogger() *zap.SugaredLogger {
	// create directory and obtain a file writer
	dir := "logs"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0774)
	}
	date := fmt.Sprintf("%d-%02d", time.Now().Year(), int(time.Now().Month()))
	localFileName := path.Join(dir, date+".log")
	var fileWriter *os.File
	i := 0
	for {
		if _, err := os.Stat(localFileName); !os.IsNotExist(err) {
			i++
			localFileName = path.Join(dir, fmt.Sprintf("%s (%d).log", date, i))
		} else {
			fileWriter, _ = os.OpenFile(localFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
			break
		}
	}

	level := zap.NewAtomicLevel()
	level.SetLevel(zapcore.DebugLevel)

	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "timestamp"
	cfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder

	stdErrWriter := zapcore.Lock(os.Stderr)
	// flush buffered logs from memory to disk every minute
	bufferWriter := &zapcore.BufferedWriteSyncer{
		WS:            fileWriter,
		FlushInterval: time.Minute,
	}
	writers := zapcore.NewMultiWriteSyncer(stdErrWriter, bufferWriter)
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(cfg), writers, level)).Sugar()

	return logger
}
