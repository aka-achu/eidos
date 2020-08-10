package eidos

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	// defaultMaxSize represents the default maximum size of the log file, which is 100 mb
	defaultMaxSize = 100
	// defaultMaxPeriod represents the maximum period of a log file to be active, which is 7 days
	defaultMaxPeriod = 7 * 24 * time.Hour
	megabyte         = 1024 * 1024
	backupTimeFormat = "2006-01-02T15-04-05.000"
	currentTime      = time.Now
)

// max return the maximum filesize
func (l *Logger) max() int64 {
	return int64(l.RotationOption.Size) * int64(megabyte)
}

// openExistingOrNewFile opens an existing log file or creates a new file.
func (l *Logger) openExistingOrNewFile() error {
	fileName := l.Filename
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
	fileName := l.Filename
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
			l.RotationOption.CompressionLevel,
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
	t := currentTime()
	if !localTime {
		t = t.UTC()
	}

	return filepath.Join(
		dir,
		fmt.Sprintf("%s-%s%s", filename[:len(filename)-len(ext)], t.Format(backupTimeFormat), ext),
	)
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
func postRotation(backupFileName string, compress bool, compressionLevel int) {
	// If compression is enabled
	if compress {
		// Get a compressed file name
		compressedFileName := fmt.Sprintf(
			"%s%s.gz",
			backupFileName[0:len(backupFileName)-len(filepath.Ext(backupFileName))],
			filepath.Ext(backupFileName),
		)
		// Compress the log file
		if err := compressLogFile(backupFileName, compressedFileName, compressionLevel); err != nil {
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
func compressLogFile(sourceFile, destinationFile string, compressionLevel int) error {
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
	gzWriter, err := gzip.NewWriterLevel(compressedFile, compressionLevel)
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
		fmt.Println("Failed to compress the file", err)
		return err
	}

	if err := gzWriter.Close(); err != nil {
		fmt.Println("Failed to close the compress writer", err)
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	// Removing the source file which is the uncompressed file
	if err := os.Remove(sourceFile); err != nil {
		return err
	}

	return nil
}

func cleanUpOldLogs(file string, compress bool, period int) {
	filename := filepath.Base(file)
	// For both compressed and uncompressed files the prefix will be
	// the base filename without extension
	prefix := filename[0 : len(filename)-len(filepath.Ext(filename))]

	// For uncompressed files the suffix will be the base file extension
	suffix := filepath.Ext(file)

	if compress {
		// For compressed files the suffix will be the extension of the compressed file
		suffix = filepath.Ext(file) + ".gz"
	}

	// get the list of all the files and folders in the log folder
	files, err := ioutil.ReadDir(filepath.Dir(file))
	if err != nil {
		return
	}

	for _, f := range files {
		// It the object is an directory, continue
		if f.IsDir() {
			continue
		}

		// a qualified rotated file will have the defined prefix and suffix
		if strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), suffix) && f.Name() != filename {
			// Parsing the time from the file name
			timeStamp, _ := time.Parse(backupTimeFormat, f.Name()[len(prefix)+1:len(f.Name())-len(suffix)])

			// Checking the age of the file, if the age is greater than the provided retention period,
			// then remove the file
			if currentTime().Sub(timeStamp.Add(-time.Second*19800)) > time.Duration(period)*time.Hour*24 {
				_ = os.Remove(filepath.Join(filepath.Dir(file), f.Name()))
			}
		}
	}
}
