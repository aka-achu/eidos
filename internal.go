package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

var defaultMaxSize = 10
var defaultMaxPeriod = 7 * 24 * time.Hour
var megabyte = 1024 * 1024
var backupTimeFormat = "2006-01-02T15-04-05.000"

func (l *Logger) max() int64 {
	return int64(l.RotationOption.Size) * int64(megabyte)
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
		backupFileName := backupName(fileName, l.RotationOption.LocalTime)
		fileMode = fileInfo.Mode()
		if err := os.Rename(fileName, backupFileName); err != nil {
			return fmt.Errorf("can't rename log file: %s", err)
		}
		if err := chown(fileName, fileInfo); err != nil {
			return err
		}
		go postRotation(
			backupFileName,
			l.RotationOption.Compress,
		)
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
	if err := l.close(); err != nil {
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

func postRotation(backupFileName string, compress bool) {
	time.Sleep(time.Second * 2)
	if compress {
		compressedFileName := backupFileName[0:len(backupFileName)-len(filepath.Ext(backupFileName))] + ".gz"
		fmt.Printf("Compression error-%v",compressLogFile(backupFileName, compressedFileName))
		callbackExecutor <- compressedFileName
	} else {
		callbackExecutor <- backupFileName
	}
}

func compressLogFile(sourceFile, destinationFile string) (err error) {
	file, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()

	fileInfo, err := os.Stat(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to stat log file: %v", err)
	}

	err = chown(destinationFile, fileInfo)
	if err != nil {
		return fmt.Errorf("failed to chown compressed log file: %v", err)
	}

	compressedFile, err := os.OpenFile(destinationFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to open compressed log file: %v", err)
	}
	defer compressedFile.Close()

	gzWriter, err := gzip.NewWriterLevel(compressedFile, gzip.BestSpeed)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			os.Remove(destinationFile)
			err = fmt.Errorf("failed to compress log file: %v", err)
		}
	}()

	if _, err := io.Copy(gzWriter, file); err != nil {
		return err
	}

	if err := gzWriter.Close(); err != nil {
		return err
	}

	if err :=file.Close(); err != nil {
		return err
	}

	if err := os.Remove(sourceFile); err != nil {
		return err
	}

	return nil
}
