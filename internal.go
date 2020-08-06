package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

var (
	// defaultMaxSize represents the default maximum size of the log file, which is 10 mb
	defaultMaxSize = 10
	// defaultMaxPeriod represents the maximum period of a log file to be active, which is 7 days
	defaultMaxPeriod = 7 * 24 * time.Hour
	megabyte = 1024 * 1024
	backupTimeFormat = "2006-01-02T15-04-05.000"
)

// max return the maximum filesize
func (l *Logger) max() int64 {
	return int64(l.RotationOption.Size) * int64(megabyte)
}

// getFilename returns the file name/path configured in the *Logger object
// if not filename is configured by the user then a default file name/path will be returned
// default file name/path is a log file with "eidos" suffix in the temp directory
func (l *Logger) getFilename() string {
	if l.Filename != "" {
		return l.Filename
	}
	return filepath.Join(os.TempDir(), filepath.Base(os.Args[0])+"-eidos.log")
}

// openExistingOrNewFile opens an existing log file or creates a new file.
func (l *Logger) openExistingOrNewFile() error {
	fileName := l.getFilename()
	fileInfo, err := os.Stat(fileName)

	// Checking for existence of the file
	if os.IsNotExist(err) {
		// Files does not exist, creating a new file
		return l.openNewFile()
	}

	if err != nil {
		return fmt.Errorf("failed to get the log file info-%v", err)
	}

	// Opening the existing file
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// If for any reason, the existing file can not be opened, open a new file
		return l.openNewFile()
	}

	// Assigning the file pointer and file size to *Logger
	l.file = file
	l.size = fileInfo.Size()
	return nil
}

// openNewFile opens a new file
func (l *Logger) openNewFile() error {
	fileName := l.getFilename()
	fileMode := os.FileMode(0666)

	// Getting the status of the requested file
	fileInfo, err := os.Stat(fileName)
	// If there is not error in file status request
	if err == nil {
		// get a backup filename
		backupFileName := backupName(fileName, l.RotationOption.LocalTime)
		fileMode = fileInfo.Mode()

		// rename file as backup file
		if err := os.Rename(fileName, backupFileName); err != nil {
			return fmt.Errorf("can't rename log file: %s", err)
		}

		if err := chown(fileName, fileInfo); err != nil {
			return err
		}

		// Trigger the post rotation thread
		go postRotation(
			backupFileName,
			l.RotationOption.Compress,
		)
	}

	// create a file to write current logs
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}

	// Assigning the file pointer and file size to *Logger
	l.file = f
	l.size = 0
	return nil
}

// backupName returns a backup name for the current file
func backupName(name string, localTime bool) string {
	dir := filepath.Dir(name)
	filename := filepath.Base(name)
	ext := filepath.Ext(filename)
	// if the localTime is true then use the system time to generate backup file name
	//if the localTime is false then use UTC time to generate backup file name
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

// rotate, rotates the currently opened log file
func (l *Logger) rotate() error {
	// Close the current log file
	if err := l.close(); err != nil {
		return err
	}

	// Open a new log file
	if err := l.openNewFile(); err != nil {
		return err
	}

	return nil
}

// close, closes the current log file
func (l *Logger) close() error {
	// If currently no file is opened
	if l.file == nil {
		return nil
	}

	// close the file, assign nil to the file pointer
	err := l.file.Close()
	l.file = nil
	return err
}

// postRotation is used to trigger callback function,
// compress the log files, if compression if enabled
func postRotation(backupFileName string, compress bool) {
	// If compression is enabled
	if compress {
		// Get a compressed file name
		compressedFileName := backupFileName[0:len(backupFileName)-len(filepath.Ext(backupFileName))] + ".gz"
		// Compress the log file
		if err := compressLogFile(backupFileName, compressedFileName); err != nil {
			// Failed to compress the log file,
			// passing the uncompressed log file path in the callback trigger channel
			callbackExecutor <- backupFileName
		} else {
			// Pass the compressed file name in the callback trigger channel
			callbackExecutor <- compressedFileName
		}
	} else {
		// Pass the backup file name in the callback trigger channel
		callbackExecutor <- backupFileName
	}
}

// compressLogFile compressed the requested log file
func compressLogFile(sourceFile, destinationFile string) error {
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

	// Using BestCompression method to compress the log files
	gzWriter, err := gzip.NewWriterLevel(compressedFile, gzip.BestCompression)
	if err != nil {
		return err
	}

	// If any error occurs, then remove the opened compressed file
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

	// Removing the source file which is the uncompressed file
	if err := os.Remove(sourceFile); err != nil {
		return err
	}

	return nil
}
