# Eidos

### Eidos is log rotation package for golang

Package eidos provides a rolling logger.
```
import "github.com/aka-achu/eidos"
```

Eidos is intended to be one part of a logging infrastructure. It is not an all-in-one solution, but instead is a pluggable component at the bottom of the logging stack that simply controls the files to which logs are written. It plays well with any logging package that can write to an io.Writer, including the standard library's log package. 

##### Assumption
Eidos assumes that only one process is writing to the output files. Using the same lumberjack configuration from multiple processes on the same machine will result in improper behavior.

##### Features
  - Filesize based log rotation
  - Inverval based log rotation
  - Log file compression
  - Retention period for rotated log files
  - Support for user defined callback function
  - Thread safe

### Objects

```go
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
```
```Logger``` is an io.WriteCloser that writes to the specified filename.

```Logger``` opens or creates the logfile on first Write. If the file exists and is less than ```Options.Size``` megabytes, eidos will open and append to that file. If the file exists and its size is >= ```Options.Size``` megabytes, the file is renamed by putting the current time in a timestamp in the name immediately before the file's extension. A new log file is then created using original filename.

Whenever a write would cause the current log file exceed MaxSize megabytes, the current file is closed, renamed, and a new log file created with the original name. Thus, the filename you give Logger is always the "current" log file. If ```compression``` is enabled in the ```Logger.RotationOption``` then, the rotated log files will be compressed using  ```gzip.BestCompression```.

### func (l *Logger) Write(p []byte) (n int, err error)
Close implements io.Write, and writes to the current logfile.

### func (l *Logger) Rotate() error
Rotate causes Logger to close the existing log file and immediately create a new one. This is a helper function for applications that want to initiate rotations outside of the normal rotation rules

### func (l *Logger) Close() error
Close implements io.Closer, and closes the current logfile.

### Callback.Execute
Callback.Execute takes a user defined function of defination func(s string). After successful rotation of the log file/ compression of log file (if compresssion if enabled), the callback function will be called with the backupfile/compressed file path as an argument. This feature can be used for some post rotation jobs like uploading the backup file to s3, etc.


### Examples
#### Using along with standard library's log package
```go
l, _ := New("/var/log/app.foobar.log",
		&Options{
			Size:      1,
			Period:    24 * time.Hour,
			Compress:  true,
			LocalTime: true,
		}, &Callback{
			Execute: func(s string) {
				fmt.Printf("Rotated file name-%s", s)
			},
		})
log.SetOutput(l)
```

#### Using along with uber-go/zap
```go
l, _ := New("/var/log/app.foobar.log",
		&Options{
			Size:      1,
			Period:    24 * time.Hour,
			Compress:  true,
			LocalTime: true,
		}, &Callback{
			Execute: func(s string) {
				fmt.Printf("Rotated file name-%s", s)
			},
		})
		
appLogger = zap.New(
    zapcore.NewCore(
        zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()),
        zapcore.AddSync(l),
        zapcore.InfoLevel,
        zap.AddCaller(),
    ).Sugar()
```




