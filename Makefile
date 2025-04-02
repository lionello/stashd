stashd: stashd.d
	ldc2 $< -of=$@

.PHONY: install
install: stashd
	install $< "${HOME}/.local/bin/"
