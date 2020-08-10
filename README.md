# Eidos [![Build Status](https://travis-ci.org/aka-achu/eidos.svg?branch=master)](https://travis-ci.org/aka-achu/eidos)

### Eidos is log rotation package for golang

Package eidos provides a rolling logger.
```
import "github.com/aka-achu/eidos"
```

Eidos is intended to be one part of a logging infrastructure. It is not an all-in-one solution, but instead is a pluggable component at the bottom of the logging stack that simply controls the files to which logs are written. It plays well with any logging package that can write to an io.Writer, including the standard library's log package. 

##### Assumption
Eidos assumes that only one process is writing to the output files. Using the same eidos configuration from multiple processes on the same machine will result in improper behavior.

##### Features
  - Filesize based log rotation
  - Interval based log rotation
  - Log file compression
  - Support for multiple compression levels
  - Retention period for rotated log files
  - Support for user defined callback function
  - Thread safe

### Objects

```go
// Logger is an io.WriteCloser that writes to the specified filename
type Logger struct {
	// Filename is the file to write logs to. Backup log files will be
	// retained in the same directory. If Filename is not given, then
	// the logs files will be written to eidos logs file and will be
	// stored in the os.TempDir() under a folder "eidos".
	Filename string `json:"filename"`

	// RotationOption specifies set of parameters for the rotating operation.
	RotationOption *Options `json:"rotation_option"`

	size            int64
	file            *os.File
	rotationTicker  *time.Ticker
	retentionTicker *time.Ticker
	mutex           sync.Mutex
}
```

```go

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

	// Compress determines if the rotated log files should be compressed is "extension.gz" format.
	// The default value of Compress in false
	Compress bool `json:"compress"`

	// CompressionLevel basically indicates the compression ratio.
	// Only three types of compression levels are supported
	// NoCompression      = 0
	// BestSpeed          = 1
	// BestCompression    = 9
	CompressionLevel int `json:"compression_level"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `json:"localtime" yaml:"localtime"`
}
```

```go

type Callback struct {
	// Execute will hold a func(string) definition which will be called when the
	// log file is being rotated and the argument to the function will be the
	// rotated/compressed file name. The user can implement some additional functionalities
	// example - upload the rotated file to s3
	Execute func(string)
}
```

```Logger``` is an io.WriteCloser that writes to the specified filename.

```Logger``` opens or creates the logfile on first Write. If the file exists and is less than ```Options.Size``` megabytes, eidos will open and append to that file. If the file exists and its size is >= ```Options.Size``` megabytes, the file is renamed by putting the current timestamp. A new log file is being created using original filename.

Whenever a write would cause the current log file exceed ```Options.Size``` megabytes, the current file is closed, renamed, and a new log file is being created with the original name. Thus, the filename you give Logger is always the "current" log file. If ```compression``` is enabled in the ```Logger.RotationOption``` then, the rotated log files will be compressed using gzip compression. The user can select the level of compression. Currently, only three compression levels are supported. 1. NoCompression 2. BestCompression 3. BestSpeed. 

### func (l *Logger) Write(p []byte) (n int, err error)
```Write``` implements ```io.Write```, and writes to the current logfile.

### func (l *Logger) Rotate() error
```Rotate``` causes Logger to close the existing log file and immediately create a new one. This is a helper function for applications that want to initiate rotations outside of the normal rotation rules.

### func (l *Logger) Close() error
```Close``` implements ```io.Closer```, and closes the current logfile.

### func New(filename string, options *Options, callback *Callback) (*Logger, error)
```New``` validates the``` eidos.options```, triggers the daemon threads and initialized the ```Logger``` object

### Daemon Threads
There are three daemon threads in eidos.
 - Period based rotation using ticker (```Logger.rotationTicker```)
     ```go
        go func() {
            for {
                select {
                case _ = <-l.rotationTicker.C:
                    l.Rotate()
                }
            }
        }()
    ```
 - Execution of custom callback function which is trigger by a channel (```callbackExecutor```)
    ```go
       go func() {
            for {
               callback.Execute(<-callbackExecutor)
            }
       }()
    ```
 - Cleaning of old log files whose retention periods has exceeded
    ```go
       go func() {
            for {
                select {
                case _ = <-l.retentionTicker.C:
                    cleanUpOldLogs(filename, options.Compress, options.RetentionPeriod)
                }
            }
       }()

    ```

### Examples
#### Using along with standard library's log package
```go
	logger, err := eidos.New("/var/log/myapp/sample.log",
		&eidos.Options{
			Size:             1000,
			Compress:         true,
			CompressionLevel: 9,
			LocalTime:        true,
			RetentionPeriod:  30,
			Period:           24 *time.Hour,
		}, &eidos.Callback{
			Execute: func(s string) {
				fmt.Println("Rotated file", s)
			},
		})
	if err != nil {
		panic(err)
	}
	log.SetOutput(logger)
```

#### Using along with uber-go/zap
```go
	logger, err := eidos.New("/var/log/myapp/sample.log",
		&eidos.Options{
			Size:             1000,
			Compress:         true,
			CompressionLevel: 9,
			LocalTime:        true,
			RetentionPeriod:  30,
			Period:           24 * time.Hour,
		}, &eidos.Callback{
			Execute: func(s string) {
				fmt.Println("Rotated file", s)
			},
		})
	if err != nil {
		panic(err)
	}
	zapcore.AddSync(logger)
```




