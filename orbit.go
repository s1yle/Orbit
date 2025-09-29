package main

import (
	"Orbit/cmd"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

type MyFormatter struct {
}

func (m *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {

	b := &bytes.Buffer{}

	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	var newLog string
	newLog = fmt.Sprintf("[%s] [%s] %s\n", timestamp, entry.Level, entry.Message)

	b.WriteString(newLog)
	return b.Bytes(), nil
}

func main() {
	log = logrus.New()

	log.SetFormatter(&MyFormatter{})

	log.SetLevel(logrus.DebugLevel)

	logDirTime := time.Now().Format("20060102_150405")
	logDirPath := filepath.Join("logs/Log_" + logDirTime)
	err := os.MkdirAll(logDirPath, 0644)
	if err != nil {
		log.Fatalf("无法创建日志文件: %v", err)
		return
	}
	logfile, err := os.OpenFile(logDirPath+"/console.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("打开日志文件失败: %v", err)
	}
	defer logfile.Close()

	log.SetOutput(io.MultiWriter(os.Stdout, logfile))

	cmd.Execute(log)
}
