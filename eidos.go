package eidos

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
		// create a new file or open an existing one
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
	// If not file is opened, return nil
	if l.file == nil {
		return nil
	}
	// Closing the opened file
	// Assigning nil to file pointer
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *Logger) Rotate() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if err := l.Close(); err != nil {
		return err
	}
	if err := l.openNewFile(); err != nil {
		return err
	}
	return nil
}
