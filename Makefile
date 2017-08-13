SRC := $(shell find . -type f -name '*.go')
CROSSARCH := amd64 386
CROSSOS := darwin linux openbsd netbsd freebsd windows

CROSS := $(foreach os,$(CROSSOS),$(foreach arch,$(CROSSARCH),dist/$(os).$(arch)))

.PHONY: reset run cross

dist/ym: $(SRC)
	@- mkdir dist 2>/dev/null
	go build -o dist/ym ./cmd/ym/*.go

run: dist/ym
	./dist/ym

install:
	go install github.com/frizinak/ym/cmd/ym

cross: $(CROSS)

$(CROSS): $(SRC)
	@- mkdir dist 2>/dev/null
	gox \
		-osarch="$(shell echo "$@" | cut -d'/' -f2- | sed 's/\./\//')" \
		-output="dist/{{.OS}}.{{.Arch}}" \
		./cmd/ym/
	if [ -f "$@.exe" ]; then mv "$@.exe" "$@"; fi

reset:
	-rm -rf dist

