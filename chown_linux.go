package eidos

import (
	"os"
	"syscall"
)

func chown(file string, info os.FileInfo) error {
	//file_info, _ := os.Stat(file)
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	f.Close()
	fileSys := info.Sys()
	return os.Chown(file, int(fileSys.(*syscall.Stat_t).Uid), int(fileSys.(*syscall.Stat_t).Gid))
}
