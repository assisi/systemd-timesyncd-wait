package main

import (
	"net"
	"os"
)

func main() {
	sync_sockname := "/run/timesyncd/time-sync.sock"

	sync_sock, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Net: "unixgram", Name: sync_sockname})
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.Write([]byte{'\n'})
		os.Exit(127)
	}

	var dat [4096]byte
	n, err := sync_sock.Read(dat[:])
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.Write([]byte{'\n'})
		os.Exit(127)
	}
	if string(dat[:n]) != "READY=1" {
		os.Exit(127)
	}
	os.Exit(0)
}
