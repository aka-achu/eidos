// +build !linux

package main

import "os"

func chown(_ string, _ os.FileInfo) error {
	return nil
}