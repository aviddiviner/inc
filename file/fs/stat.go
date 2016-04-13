// +build linux darwin dragonfly freebsd openbsd netbsd solaris

package fs

import (
	"os"
	"syscall"
)

func sysStat(fi os.FileInfo) (fstat *FileStat_t, err error) {
	sys, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return // TODO: error
	}
	fstat = &FileStat_t{
		Uid: int(sys.Uid),
		Gid: int(sys.Gid),
		// Atime: statAtime(sys), // TODO: use these?
		// Ctime: statCtime(sys),
	}
	return
}
