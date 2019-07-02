SRC := $(shell find . -type f -name '*.go'; echo go.mod)
TAGS := $(shell pkg-config mpv || echo nolibmpv)
VERSION := $(shell git describe)
BUILD_FLAGS := -ldflags "-X main.version=$(VERSION)" -tags '$(TAGS)'
BINS := ym ym-cache ym-files
OS := linux darwin windows
CROSS := $(foreach bin,$(BINS),$(foreach os,$(OS),$(if $(findstring windows,$(os)),dist/$(bin).$(os).exe,dist/$(bin).$(os))))
NATIVE := $(foreach bin,$(BINS),dist/$(bin).native)
RELEASE := $(foreach bin,$(BINS),$(foreach os,$(OS),$(if $(findstring windows,$(os)),dist/release/$(os)/$(bin)-nolibmpv.exe,dist/release/$(os)/$(bin)-nolibmpv)))

.PHONY: all
all: $(NATIVE)

dist/ym.native: $(SRC) | dist
	go build $(BUILD_FLAGS) -o $@ ./cmd/ym/*.go

dist/ym-%.native: $(SRC) | dist
	go build $(BUILD_FLAGS) -o $@ ./cmd/$(shell basename "$@" | cut -d'.' -f1)/*.go

dist/ym%: $(SRC) | dist
	gox $(BUILD_FLAGS) \
		-tags 'nolibmpv' \
		-osarch="$(shell echo "$@" | cut -d'.' -f2 | rev | cut -d'-' -f1 | rev)/amd64" \
		-output="$(shell echo "$@" | cut -d'.' -f-2)" ./cmd/$(shell basename "$@" | cut -d'.' -f1)

dist:
	@mkdir dist 2>/dev/null || true

dist/release/%: cross
	@mkdir dist/release 2>/dev/null || true
	@os="$$(dirname "$*")"; \
	   mkdir "$$(dirname "$@")" 2>/dev/null; \
	   src="$$(basename "$*" | sed "s/-nolibmpv/.$$os/")"; \
	   cp "dist/$$src" "$@";

.PHONY: run
run: dist/ym.native
	./dist/ym.native

.PHONY: install
install:
	@for i in $(BINS); do \
		go install $(BUILD_FLAGS) github.com/frizinak/ym/cmd/$$i; \
		done

.PHONY: complete
complete:
	go build -i -buildmode=default -tags '$(TAGS)' -o /dev/null ./cmd/ym/*.go

.PHONY: cross
cross: $(CROSS)

.PHONY: release
release: $(RELEASE)
	@if ! echo "$(TAGS)" | grep nolibmpv > /dev/null; then \
		$(MAKE) all; \
		for i in dist/ym*.native; do cp "$$i" "dist/release/linux/$$(basename $$i | cut -d '.' -f1)" ; done; \
	fi
	@echo -e "\033[1;30;42m Release: $(VERSION) \033[0m"


.PHONY: reset
reset:
	-rm -rf dist

