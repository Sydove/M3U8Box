package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	initOnce sync.Once
	initErr  error
	logFile  *os.File

	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
)

func Init() error {
	initOnce.Do(func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			initErr = err
			return
		}

		logDir := filepath.Join(homeDir, "m2u8box")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			initErr = err
			return
		}

		logPath := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			initErr = err
			return
		}

		logFile = file
		writer := io.MultiWriter(os.Stdout, logFile)
		infoLogger = log.New(writer, "INFO ", log.LstdFlags)
		warnLogger = log.New(writer, "WARN ", log.LstdFlags)
		errorLogger = log.New(writer, "ERROR ", log.LstdFlags)
	})

	return initErr
}

func Close() error {
	if logFile == nil {
		return nil
	}
	return logFile.Close()
}

func FileWriter() io.Writer {
	if logFile == nil {
		return io.Discard
	}
	return logFile
}

func Infof(format string, args ...any) {
	if infoLogger == nil {
		log.Printf("INFO "+format, args...)
		return
	}
	infoLogger.Printf(format, args...)
}

func Warnf(format string, args ...any) {
	if warnLogger == nil {
		log.Printf("WARN "+format, args...)
		return
	}
	warnLogger.Printf(format, args...)
}

func Errorf(format string, args ...any) {
	if errorLogger == nil {
		log.Printf("ERROR "+format, args...)
		return
	}
	errorLogger.Printf(format, args...)
}
