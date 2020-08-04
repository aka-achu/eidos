package main

import (
	"fmt"
	"io"
)

// Implements io.WriteCloser
var _ io.WriteCloser = (*Logger)(nil)

func (l *Logger) Write(p []byte) (n int, err error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	writeRequestLength := int64(len(p))
	maxFileSize := l.max()

	// If the requested write length exceeds the maximum file size then return err
	if writeRequestLength > maxFileSize {
		return 0,
			fmt.Errorf(
				"write request size %d exceed max file size %d", writeRequestLength, maxFileSize,
			)
	}
	if l.file == nil {
		if err := l.openExistingOrNewFile(); err != nil {
			return 0,err
		}
	}

	if l.size+writeRequestLength > l.max() {
		if err := l.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = l.file.Write(p)
	l.size += int64(n)
	return n,err
}

// Close implements io.Closer
// It closed current log file if it's open
func (l *Logger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.close()
}

func (l *Logger) Rotate() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.rotate()
}
