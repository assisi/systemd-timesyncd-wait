Proper time-sync.target support for systemd-timesyncd

This package essentially just works around
  https://github.com/systemd/systemd/issues/5097

systemd.special(7) tells us that "All services where correct time is
essential should be ordered after [time-sync.target]".  However,
systemd-timesyncd allows time-sync.target to be reached before
timesyncd has actually synchronized the time.  This is because it
sends READY=1 as soon as the daemon has initialized, rather that
waiting until it has successfully synchronized to an NTP server.

It would be trivial to patch timesyncd to wait, but that would
introduce some other problems.

So, I'm introducing systemd-timesyncd-wait.  It is a service that
listens for messages from systemd-timesyncd, and block until it sees a
message indicating that systemd-timesyncd has synchronized the time.

### Requirements

	go > 1.4
	make


### Installation

Clone the repo and execute:

	make && make install
