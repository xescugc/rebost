package main

import (
	"log"
	"os"
)

type Logger struct {
	Error *log.Logger
	Info  *log.Logger
	Warn  *log.Logger
}

var logger *Logger = NewLogger()

//TODO: accept a config object to configure logger
func NewLogger() *Logger {
	return &Logger{
		Error: log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime),
		Info:  log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime),
		Warn:  log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime),
	}
}

func (l *Logger) info(v ...interface{}) {
	l.Info.Println(v)
}

func (l *Logger) warn(v ...interface{}) {
	l.Warn.Println(v)
}

func (l *Logger) error(v ...interface{}) {
	l.Error.Println(v)
}
