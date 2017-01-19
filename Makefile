files.out.all += systemd-timesyncd-wait systemd-timesyncd-wrap
files.sys.all += /usr/lib/systemd/systemd-timesyncd-wait
files.sys.all += /usr/lib/systemd/systemd-timesyncd-wrap
files.sys.all += /usr/lib/systemd/system/systemd-timesyncd-wait.socket
files.sys.all += /usr/lib/systemd/system/systemd-timesyncd-wait.service
files.sys.all += /usr/lib/systemd/system/systemd-timesyncd.service.d/wait.conf

outdir = .
srcdir = .
all: $(addprefix $(outdir)/,$(files.out.all))
clean:
	rm -f -- $(addprefix $(outdir)/,$(files.out.all))
install: $(addprefix $(DESTDIR),$(files.sys.all))
.PHONY: all clean install

$(outdir)/%: $(srcdir)/%.go
	go build -o $@ $<

$(DESTDIR)/usr/lib/systemd/%: $(outdir)/%
	install -DTm755 $< $@
$(DESTDIR)/usr/lib/systemd/system/%: $(srcdir)/%
	install -DTm644 $< $@
$(DESTDIR)/usr/lib/systemd/system/systemd-timesyncd.service.d/wait.conf: systemd-timesyncd.service.d-wait.conf
	install -DTm644 $< $@
