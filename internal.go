package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var defaultMaxSize = 10
var megabyte = 1024 * 1024
var backupTimeFormat = "2006-01-02T15-04-05.000"

func (l *Logger) max() int64 {
	if l.RollingOption.Size == 0 {
		return int64(defaultMaxSize * megabyte)
	}
	return int64(l.RollingOption.Size) * int64(megabyte)
}

func (l *Logger) getFilename() string {
	if l.Filename != "" {
		return l.Filename
	}
	return filepath.Join(os.TempDir(), filepath.Base(os.Args[0])+"-eidos.log")
}

func (l *Logger) openExistingOrNewFile() error {
	fileName := l.getFilename()
	fileInfo, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return l.openNewFile()
	}
	if err != nil {
		return fmt.Errorf("failed to get the log file info-%v", err)
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return l.openNewFile()
	}
	l.file = file
	l.size = fileInfo.Size()
	return nil
}

func (l *Logger) openNewFile() error {
	fileName := l.getFilename()
	fileMode := os.FileMode(0666)
	fileInfo, err := os.Stat(fileName)
	if err == nil {
		backupFileName := backupName(fileName, l.RollingOption.LocalTime)
		fileMode = fileInfo.Mode()
		if err := os.Rename(fileName, backupFileName ); err != nil {
			return fmt.Errorf("can't rename log file: %s", err)
		}
		if err := chown(fileName, fileInfo); err != nil {
			return err
		}
		go l.RollingOption.Callback(backupFileName)
	}
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}
	l.file = f
	l.size = 0
	return nil

}

func backupName(name string, localTime bool) string {
	dir := filepath.Dir(name)
	filename := filepath.Base(name)
	ext := filepath.Ext(filename)
	t := time.Now()
	if !localTime {
		t = t.UTC()
	}
	return filepath.Join(
		dir,
		fmt.Sprintf("%s-%s%s", filename[:len(filename)-len(ext)], t.Format(backupTimeFormat), ext),
	)
}

func chown(_ string, _ os.FileInfo) error {
	return nil
}

func (l *Logger) rotate() error {
	if err := l.close(); err != nil  {
		return err
	}
	if err := l.openNewFile(); err != nil {
		return err
	}
	return nil
}

func (l *Logger) close() error {
	if l.file == nil {
		return nil
	}
	// Closing the opened file
	// Assigning nil to file pointer
	err := l.file.Close()
	l.file = nil
	return err
}
