package main

import (
	"os"
	"sync"
	"time"
)

// Logger is an ip.WriteCloser that writes to the specified filename
type Logger struct {
	// Filename is the file to write logs to. Backup log files will be
	// retained in the same directory. If Filename is not given, then
	// the logs files will be written to eidos logs file and will be
	// stored in the os.TempDir().
	Filename string `json:"filename"`

	// RotationOption specifies set of parameters for the rotating operation.
	RotationOption *Options `json:"rotation_option"`

	size  int64
	file  *os.File
	ticker *time.Ticker
	tick chan bool
	mutex sync.Mutex
}

type Options struct {

	// Size is the maximum size in megabytes of the log file before it gets
	// rotated. The default Size is 100 megabyte
	Size int `json:"size"`

	// Period is the maximum age of the log file before it gets rotated.
	// The default Period of the log file is 7 days
	Period time.Duration `json:"period"`

	// RetentionPeriod is the maximum number of days to retain old log files based
	// on the timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	RetentionPeriod int `json:"retention_period"`

	// Compress determines if the rotated log files should be compressed. The default
	// value of Compress in false
	Compress bool `json:"compress"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `json:"localtime" yaml:"localtime"`
}

type Callback struct {
	// Execute will hold a func(string) definition which will be called when the
	// log file is being rotated and the argument to the function will be the
	// rotated/compressed file name. The user can implement some additional functionalities
	// example - upload the rotated file to s3
	Execute func(string)
}