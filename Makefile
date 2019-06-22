SRC := $(shell find . -type f -name '*.go')
TAGS := $(shell pkg-config mpv || echo nolibmpv)
VERSION := $(shell git describe)
BUILD_FLAGS := -ldflags "-X main.version=$(VERSION)" -tags '$(TAGS)'

dist/ym-native: $(SRC)
	@- mkdir dist 2>/dev/null
	go build $(BUILD_FLAGS) -o $@ ./cmd/ym/*.go

dist/ym-%: $(SRC)
	@- mkdir dist 2>/dev/null
	gox $(BUILD_FLAGS) \
		-tags 'nolibmpv' \
		-osarch="$(shell echo "$*" | cut -d'.' -f1)/amd64" \
		-output="$(shell echo "$@" | cut -d'.' -f1)" ./cmd/ym/

.PHONY: release
release: | reset cross
	@for i in dist/*; do mv "$$i" "$$i-nolibmpv"; done
	@if ! echo "$(TAGS)" | grep nolibmpv > /dev/null; then \
		$(MAKE) dist/ym-native && mv dist/ym-native dist/ym-linux; \
	fi
	@echo -e "\033[1;30;42m Release: $(VERSION) \033[0m"

.PHONY: run
run: dist/ym-native
	./dist/ym-native

.PHONY: install
install:
	go install $(BUILD_FLAGS) github.com/frizinak/ym/cmd/ym

.PHONY: complete
complete:
	go build -i -buildmode=default -tags '$(TAGS)' -o /dev/null ./cmd/ym/*.go

.PHONY: cross
cross: dist/ym-windows.exe dist/ym-linux dist/ym-darwin

.PHONY: reset
reset:
	-rm -rf dist

