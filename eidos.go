package main

import (
	"fmt"
	"io"
	"time"
)

// Implements io.WriteCloser
var _ io.WriteCloser = (*Logger)(nil)


func New(filename string, options *Options) *Logger {
	if options.Callback == nil {
		options.Callback = func(s string) {}
	}

	if options.Size == 0 {
		options.Size = defaultMaxSize
	}

	if options.Period == time.Duration(0) {
		options.Period = defaultMaxPeriod
	}
	l := &Logger{
		Filename:      filename,
		RollingOption: options,
	}

	l.ticker = time.NewTicker(options.Period)
	l.tick = make(chan bool)

	go func() {
		for {
			select {
			case <-l.tick:
				return
			case _ = <-l.ticker.C:
				l.Rotate()
			}
		}
	}()

	return l
}

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
