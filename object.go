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

	// RollingOption specifies set of parameters for the rolling operation.
	RollingOption *Options `json:"rolling_option"`

	// RolledFile is a channel of string which will be used to send the
	// path of the rolled file. If the compression is enabled in the options,
	// the CompressedFile channel will be used to send path of rolled
	// compressed log file.
	RolledFile <-chan string `json:"rolled_file"`

	// CompressedFile is a channel of string which will be used to send the
	// path of the rolled compressed log file. If the compression id disabled
	// in the options,  RolledFile channel will be used to send the path of
	// rolled file.
	CompressedFile <-chan string `json:"compressed_file"`

	size  int64
	file  *os.File
	mutex sync.Mutex
}

type Options struct {

	// Size is the maximum size in megabytes of the log file before it gets
	// rolled. The default Size is 100 megabyte
	Size int `json:"size"`

	// Period is the maximum age of the log file before it gets rolled.
	// The default Period of the log file is 7 days
	Period time.Duration `json:"period"`

	// RetentionPeriod is the maximum number of days to retain old log files based
	// on the timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	RetentionPeriod int `json:"retention_period"`

	// Compress determines if the rolled log files should be compressed. The default
	// value of Compress in false
	Compress bool `json:"compress"`

	// Callback will hold a func(string) definition which will be called when the
	// log file is being rolled and the argument to the function will be the
	// rolled/compressed file name. The user can implement some additional functionalities
	// example - upload the rolled file to s3
	Callback func(string)

	// CleanUpCallback will hold an internal cleanup function definition which will
	// clean old log file and some other post rolling operations
	CleanUpCallback func()
}
