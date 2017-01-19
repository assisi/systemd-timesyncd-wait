package main

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"
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

	user, err := user.Lookup("systemd-timesync")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(127)
	}
	uid, _ := strconv.Atoi(user.Uid)
	gid, _ := strconv.Atoi(user.Gid)

	umask := syscall.Umask(0577)
	fake_sock, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Net: "unixgram", Name: fake_sockname})
	syscall.Umask(umask)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(127)
	}
	os.Chown(fake_sockname, uid, gid)

	os.Setenv("NOTIFY_SOCKET", fake_sockname)
	proc, err := os.StartProcess(os.Args[1], os.Args[1:], &os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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
				fmt.Fprintln(os.Stderr, err)
				break
			}
			if flags&syscall.MSG_TRUNC != 0 {
				fmt.Fprintln(os.Stderr, "Received notify message exceeded maximum size. Ignoring.")
				continue
			}
			if !synced {
				for _, line := range strings.Split(string(dat[:n]), "\n") {
					if strings.HasPrefix(line, "STATUS=Synchronized") {
						err = sendmsg([]byte(line), nil, sync_sockname)
						if err != nil {
							fmt.Fprintln(os.Stderr, err)
						}
						synced = true
						break
					}
				}
			}
			err = sendmsg(dat[:n], oob[:oobn], real_sockname)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		wg.Done()
	}()

	state, err := proc.Wait()
	_ = fake_sock.Close()
	wg.Wait()

	if err != nil {
		fmt.Println(os.Stderr, err)
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
