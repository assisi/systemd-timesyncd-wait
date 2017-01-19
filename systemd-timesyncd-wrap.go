package main

import (
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
)

func socketUnixgram(name string) (*net.UnixConn, error) {
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	conn, err := net.FileConn(os.NewFile(uintptr(fd), name))
	if err != nil {
		return nil, err
	}
	unixConn := conn.(*net.UnixConn)
	return unixConn, nil
}

func sendmsg(dat, oob []byte, sockname string) error {
	writer, err := socketUnixgram(sockname)
	if err != nil {
		return err
	}
	_, _, err = writer.WriteMsgUnix(dat, oob, &net.UnixAddr{Net: "unixgram", Name: sockname})
	return err
}

func main() {
	sync_sockname := "/run/timesyncd/time-sync.sock"
	fake_sockname := "/run/timesyncd/notify.sock"
	real_sockname := os.Getenv("NOTIFY_SOCKET")

	fake_sock, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Net: "unixgram", Name: fake_sockname})
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.Write([]byte{'\n'})
		os.Exit(127)
	}

	os.Setenv("NOTIFY_SOCKET", fake_sockname)
	proc, err := os.StartProcess(os.Args[1], os.Args[1:], &os.ProcAttr{})
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.Write([]byte{'\n'})
		os.Exit(127)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		synced := false
		var dat [4096]byte
		oob := make([]byte, syscall.CmsgSpace(syscall.SizeofUcred)+syscall.CmsgSpace(8*768))
		for {
			n, oobn, flags, _, err := fake_sock.ReadMsgUnix(dat[:], oob[:])
			if err != nil {
				break
			}
			if flags&syscall.MSG_TRUNC != 0 {
				continue
			}
			if !synced {
				for _, line := range strings.Split(string(dat[:n]), "\n") {
					if strings.HasPrefix(line, "STATUS=Synchronized") {
						_ = sendmsg([]byte(line), nil, sync_sockname)
						synced = true
						break
					}
				}
			}
			_ = sendmsg(dat[:n], oob[:oobn], real_sockname)
		}
		wg.Done()
	}()

	state, err := proc.Wait()
	_ = fake_sock.Close()
	wg.Wait()

	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.Write([]byte{'\n'})
		os.Exit(127)
	}
	status := state.Sys().(syscall.WaitStatus)
	if status.Exited() {
		os.Exit(status.ExitStatus())
	}
	if status.Signaled() {
		self, _ := os.FindProcess(os.Getpid())
		self.Signal(status.Signal())
	}
}
