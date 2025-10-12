package logger

import (
	"io"
	"log"
	"os"
)

var l log.Logger

var level int

// array of log levels to match the config file and the const
var loglevelstr = []string{"Err", "Warn", "Inf", "Deb"}

const (
	// ErrorLevel level. Used for errors that should definitely be noted.
	Err = iota
	// WarningLevel level. Non-critical entries that deserve eyes.
	Warn
	// InfoLevel level. General operational entries about what's going on inside the application.
	Inf
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	Deb
)

// Init initializes the logger
func Init(file string, llevel string) {

	for i, v := range loglevelstr {
		if v == llevel {
			level = i
		}
	}

	fileName := file
	var err error
	logFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}

	multi := io.MultiWriter(logFile, os.Stdout)
	l.SetOutput(multi)
	//l.SetOutput(logFile)
	//l.SetFlags(log.Lshortfile | log.LstdFlags)
	l.SetFlags(log.LstdFlags)
}

func Error(msg ...interface{}) {
	if level >= Err {
		l.Println("ERROR:", msg)
	}
}

func Warning(msg ...interface{}) {
	if level >= Warn {
		l.Println("WARNING:", msg)
	}
}

func Info(msg ...interface{}) {
	if level >= Inf {
		l.Println("INFO:", msg)
	}
}

func Debug(msg ...interface{}) {
	if level >= Deb {
		l.Println("DEBUG:", msg)
	}
}
