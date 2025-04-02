stashd: stashd.go
	go build -o $@ $<

.PHONY: install
install: stashd
	install $< "${HOME}/.local/bin/"
