package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Implements io.WriteCloser
var _ io.WriteCloser = (*Logger)(nil)
// callbackExecutor acts as a communication pipeline in between the main thread and the callback daemon thread
var callbackExecutor chan string

// New initialized the *Logger object and run daemons
func New(filename string, options *Options, callback *Callback) (*Logger, error) {
	// If the callback.Execute does not contain any functions,
	// initialize with a empty method.
	if callback.Execute == nil {
		callback.Execute = func(s string) {}
	}

	// If the options does not have any .Size value,
	// initialize with defaultMaxSize.
	if options.Size == 0 {
		options.Size = defaultMaxSize
	}

	// If the option does not have any .Period value,
	// initialize with defaultMaxPeriod.
	if options.Period == time.Duration(0) {
		options.Period = defaultMaxPeriod
	}

	// Initializing a Logger object
	l := &Logger{
		Filename:       filename,
		RotationOption: options,
	}

	// Initializing callbackExecutor channel
	callbackExecutor = make(chan string)

	// Creating directories for the log files
	//todo check the requested path of directories exists of not
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return nil, err
	}

	// Initializing a ticker of interval options.Period
	l.ticker = time.NewTicker(options.Period)
	l.tick = make(chan bool)

	// Running daemon go-routine for period based
	// rotation of log files
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

	// Running daemon go-routine for execution of callback method
	// callback.Execute waits to receive data from callbackExecutor
	// channel which is used to send rotated filename / compressed
	// filename from the postRotation thread to daemon thread.
	go func(){
		for {
			callback.Execute(<-callbackExecutor)
		}
	}()

	return l, nil
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

	// If the file pointer in the Logger object is nil then open a new/existing log file
	if l.file == nil {
		if err := l.openExistingOrNewFile(); err != nil {
			return 0,err
		}
	}

	// If writing the requested data to the file will make the file size
	// exceed the max allowed filesize, then rotate the current file.
	if l.size+writeRequestLength > l.max() {
		if err := l.rotate(); err != nil {
			return 0, err
		}
	}

	// Write the requested data to the file
	n, err = l.file.Write(p)

	// Increase the file size by request content length
	l.size += int64(n)

	return n,err
}

// Close implements io.Closer
// It closes current log file if it's open
func (l *Logger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.close()
}

// Rotate, rotates the current file,
// the file will be compressed if the
// compression option is turned on.
func (l *Logger) Rotate() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.rotate()
}
