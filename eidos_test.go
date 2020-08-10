package eidos

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func equals(src, des interface{}, t *testing.T, message string) {
	if !reflect.DeepEqual(src, des) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("Msg-%v. File-%v Line-%v \n", message, file, line)
		t.FailNow()
	}
}

func clean(dir string) error {
	return os.RemoveAll(dir)
}

func randStringBytes(n int) string {
	var letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func TestNewEmpty(t *testing.T) {
	// test for default value initialization
	logger, err := New("", &Options{}, &Callback{})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	equals(
		err,
		nil,
		t,
		"Failed to initialize the *Logger object",
	)
	equals(
		logger.RotationOption.Size,
		defaultMaxSize,
		t,
		"Invalid default value initialization",
	)
	equals(
		logger.RotationOption.Period,
		defaultMaxPeriod,
		t,
		"Invalid default value initialization",
	)
	equals(
		logger.Filename,
		filepath.Join(os.TempDir(), "eidos_logs", filepath.Base(os.Args[0])+"-eidos.log"),
		t,
		"Invalid default value initialization",
	)

	// Checking the creation of the default log directory path
	_, err = os.Stat(filepath.Dir(logger.Filename))
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Failed to create the default log directory path",
	)

}

func TestNew(t *testing.T) {
	// test for default value initialization
	logger, err := New("var/log/myapp.sample.log", &Options{
		Size:             100,
		Period:           24 * 7 * time.Hour,
		RetentionPeriod:  30,
		Compress:         true,
		CompressionLevel: 100, // It should be initialized to 1
		LocalTime:        true,
	}, &Callback{
		Execute: func(s string) {
			fmt.Printf("Rotated File- %s", s)
		},
	})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	equals(
		err,
		nil,
		t,
		"Failed to initialize the *Logger object",
	)

	equals(
		logger.RotationOption.CompressionLevel,
		0,
		t,
		"Failed to validate the *Logger.RotationOption.CompressionLevel value",
	)

	// Checking the creation of the default log directory path
	_, err = os.Stat(filepath.Dir(logger.Filename))
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Failed to create the requested log directory path",
	)
}

func TestLogger_Close(t *testing.T) {
	logger, _ := New("", &Options{}, &Callback{})
	defer func() {
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()
	equals(logger.Close(), nil, t, "Failed to close the Logger")
}

// Test for max length exceeding write request
func TestLogger_Write_Max(t *testing.T) {
	logger, _ := New("", &Options{
		Size: 1,
	}, &Callback{})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	log.SetOutput(logger)

	// Getting a random string whose length is greater than the max size of the file
	body := randStringBytes(1024*1024 + 1)
	// writing a log whose size is greater than max file size
	log.Println(body)
	// The log file will not be created. Checking status of the log file
	_, err := os.Stat(logger.Filename)
	equals(
		os.IsNotExist(err),
		true,
		t,
		"Error. The requested write length exceed the max file size. The file should not be created",
	)
}

func TestLogger_Write_New_Existing(t *testing.T) {
	logger, _ := New("", &Options{
		Size: 1,
	}, &Callback{})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	log.SetOutput(logger)

	body := randStringBytes(1024)
	log.Println(body)

	// Checking for the existence of the log file
	fileInfo, err := os.Stat(logger.Filename)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The log file should be created",
	)

	// Validating the filesize with the written content size
	// requested content size will be less than the actual file size because of the log prefixes
	equals(
		fileInfo.Size(),
		int64(1045),
		t,
		"Error. The file size does not match to the expected file size",
	)
	// Closing the logger to reinitialize the object
	_ = logger.close()

	// Repeating the same steps for validating opening of the same log file
	logger, _ = New("", &Options{
		Size: 1,
	}, &Callback{})
	log.SetOutput(logger)

	log.Println(body)

	// Checking for the existence of the log file
	fileInfo, err = os.Stat(logger.Filename)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The log file should be created",
	)

	// Validating the filesize with the written content size
	// requested content size will be less than the actual file size because of the log prefixes
	equals(
		fileInfo.Size(),
		int64(2090),
		t,
		"Error. The file size does not match to the expected file size",
	)
}

func TestLogger_Rotate_Auto_Size(t *testing.T) {

	var rotateCh = make(chan string, 1)
	logger, _ := New("", &Options{
		Size: 1,
	}, &Callback{
		Execute: func(s string) {
			rotateCh <- s
		},
	})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	log.SetOutput(logger)

	body := randStringBytes(1024)

	for index := 0; index < 1024; index++ {
		log.Println(body)
	}

	// Checking for the existence of the rotated log file
	fileInfo, err := os.Stat(<-rotateCh)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The rotated log file should be created",
	)

	// Checking for the existence of the original file
	fileInfo, err = os.Stat(logger.Filename)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The original log file should be present",
	)

	// Validating the rotated filesize with the written content size
	if fileInfo.Size() >= 1024*1024 {
		t.Logf("Error- The roatated file size is greater than expected")
		t.FailNow()
	}
}

func TestLogger_Rotate_Manual(t *testing.T) {

	var rotateCh = make(chan string, 1)
	logger, _ := New("", &Options{
		Size: 1,
	}, &Callback{
		Execute: func(s string) {
			rotateCh <- s
		},
	})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	log.SetOutput(logger)

	body := randStringBytes(1024)
	log.Println(body)

	// Rotating the log file manually
	equals(
		logger.Rotate(),
		nil,
		t,
		"Error. Failed to rotate the log file manually",
	)

	// Checking for the existence of the rotated log file
	_, err := os.Stat(<-rotateCh)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The rotated log file should be created",
	)

	// Checking for the existence of the original file
	_, err = os.Stat(logger.Filename)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The original log file should be present",
	)
}

func TestLogger_Rotate_Auto_Size_Compress(t *testing.T) {

	var rotateCh = make(chan string, 1)
	logger, _ := New("", &Options{
		Size:             1,
		Compress:         true,
		CompressionLevel: 9,
	}, &Callback{
		Execute: func(s string) {
			rotateCh <- s
		},
	})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	log.SetOutput(logger)

	body := randStringBytes(1024)

	for index := 0; index < 1024; index++ {
		log.Println(body)
	}

	//todo check the existence logic of files

	// Checking for the existence of the rotated log file
	_, err := os.Stat(<-rotateCh)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The rotated log file should be created",
	)

	// Checking for the existence of the original file
	_, err = os.Stat(logger.Filename)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The original log file should be present",
	)
}

func TestLogger_Rotate_Manual_Compress(t *testing.T) {

	var rotateCh = make(chan string, 1)
	logger, _ := New("", &Options{
		Size:             1,
		Compress:         true,
		CompressionLevel: 9,
	}, &Callback{
		Execute: func(s string) {
			rotateCh <- s
		},
	})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	log.SetOutput(logger)

	body := randStringBytes(1024)
	log.Println(body)

	// Rotating the log file manually
	equals(
		logger.Rotate(),
		nil,
		t,
		"Error. Failed to rotate the log file manually",
	)

	// Checking for the existence of the rotated log file
	_, err := os.Stat(<-rotateCh)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The rotated compressed log file should be created",
	)

	// Checking for the existence of the original file
	_, err = os.Stat(logger.Filename)
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. The original log file should be present",
	)
}

func TestLogger_Retention_Compressed(t *testing.T) {

	_ = os.MkdirAll(filepath.Join(os.TempDir(), "eidos_logs"), 0755)

	dayOldFile := fmt.Sprintf(
		"%s-eidos-%s.log.gz",
		os.Args[0],
		time.Now().Add(-24*time.Hour).Format(backupTimeFormat),
	)

	weekOldFile := fmt.Sprintf(
		"%s-eidos-%s.log.gz",
		os.Args[0],
		time.Now().Add(-7*24*time.Hour).Format(backupTimeFormat),
	)

	monthOldFile := fmt.Sprintf(
		"%s-eidos-%s.log.gz",
		os.Args[0],
		time.Now().Add(-31*24*time.Hour).Format(backupTimeFormat),
	)

	f, _ := os.OpenFile(
		filepath.Join(
			os.TempDir(),
			"eidos_logs",
			filepath.Base(
				dayOldFile,
			),
		), os.O_CREATE, 0655)
	_ = f.Close()
	f, _ = os.OpenFile(
		filepath.Join(
			os.TempDir(),
			"eidos_logs",
			filepath.Base(
				weekOldFile,
			),
		), os.O_CREATE, 0655)
	_ = f.Close()
	f, _ = os.OpenFile(
		filepath.Join(
			os.TempDir(),
			"eidos_logs",
			filepath.Base(
				monthOldFile,
			),
		), os.O_CREATE, 0655)
	_ = f.Close()

	logger, _ := New("", &Options{
		RetentionPeriod: 10,
		Compress:        true,
	}, &Callback{})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	log.SetOutput(logger)

	log.Println(randStringBytes(1024))
	// Waiting for the clean up thread to clean the fake old compressed log files
	time.Sleep(time.Second * 5)

	// Validating the existence of the fake files
	_, err := os.Stat(filepath.Join(
		os.TempDir(),
		"eidos_logs",
		filepath.Base(
			dayOldFile,
		),
	))
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. Day old compressed file should be present in the log folder",
	)

	_, err = os.Stat(filepath.Join(
		os.TempDir(),
		"eidos_logs",
		filepath.Base(
			weekOldFile,
		),
	))
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. Week old compressed file should be present in the log folder",
	)

	_, err = os.Stat(filepath.Join(
		os.TempDir(),
		"eidos_logs",
		filepath.Base(
			monthOldFile,
		),
	))
	equals(
		os.IsNotExist(err),
		true,
		t,
		"Error. Month old compressed file should not be present in the log folder",
	)
}

func TestLogger_Retention_UnCompressed(t *testing.T) {

	_ = os.MkdirAll(filepath.Join(os.TempDir(), "eidos_logs"), 0755)

	dayOldFile := fmt.Sprintf(
		"%s-eidos-%s.log",
		os.Args[0],
		time.Now().Add(-24*time.Hour).Format(backupTimeFormat),
	)

	weekOldFile := fmt.Sprintf(
		"%s-eidos-%s.log",
		os.Args[0],
		time.Now().Add(-7*24*time.Hour).Format(backupTimeFormat),
	)

	monthOldFile := fmt.Sprintf(
		"%s-eidos-%s.log",
		os.Args[0],
		time.Now().Add(-31*24*time.Hour).Format(backupTimeFormat),
	)

	f, _ := os.OpenFile(
		filepath.Join(
			os.TempDir(),
			"eidos_logs",
			filepath.Base(
				dayOldFile,
			),
		), os.O_CREATE, 0655)
	_ = f.Close()
	f, _ = os.OpenFile(
		filepath.Join(
			os.TempDir(),
			"eidos_logs",
			filepath.Base(
				weekOldFile,
			),
		), os.O_CREATE, 0655)
	_ = f.Close()
	f, _ = os.OpenFile(
		filepath.Join(
			os.TempDir(),
			"eidos_logs",
			filepath.Base(
				monthOldFile,
			),
		), os.O_CREATE, 0655)
	_ = f.Close()

	logger, _ := New("", &Options{
		RetentionPeriod: 10,
		Compress:        false,
	}, &Callback{})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	log.SetOutput(logger)
	// Waiting for the clean up thread to clean the fake old compressed log files
	time.Sleep(time.Second * 5)

	// Validating the existence of the fake files
	_, err := os.Stat(filepath.Join(
		os.TempDir(),
		"eidos_logs",
		filepath.Base(
			dayOldFile,
		),
	))
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. Day old compressed file should be present in the log folder",
	)

	_, err = os.Stat(filepath.Join(
		os.TempDir(),
		"eidos_logs",
		filepath.Base(
			weekOldFile,
		),
	))
	equals(
		os.IsNotExist(err),
		false,
		t,
		"Error. Week old compressed file should be present in the log folder",
	)

	_, err = os.Stat(filepath.Join(
		os.TempDir(),
		"eidos_logs",
		filepath.Base(
			monthOldFile,
		),
	))
	equals(
		os.IsNotExist(err),
		true,
		t,
		"Error. Month old compressed file should not be present in the log folder",
	)
}

func TestLogger_Rotate_Auto_Period_Compress(t *testing.T) {

	var rotateCh = make(chan string, 10)

	go func() {
		for file := range rotateCh {
			_, err := os.Stat(file)
			equals(
				os.IsNotExist(err),
				false,
				t,
				"Error. The rotated compressed log file should be present",
			)
		}
	}()
	logger, _ := New("", &Options{
		Compress:         true,
		CompressionLevel: 9,
		Period:           time.Second * 2,
	}, &Callback{
		Execute: func(s string) {
			rotateCh <- s
		},
	})

	defer func() {
		// Closing the logger to clean up the log directory
		_ = logger.close()
		// Cleaning up the log directory
		_ = clean(filepath.Dir(logger.Filename))
	}()

	log.SetOutput(logger)
	log.Println(randStringBytes(1024))
	time.Sleep(time.Second * 3)
	logger.rotationTicker.Stop()
	close(rotateCh)
	time.Sleep(time.Second * 1)

}
