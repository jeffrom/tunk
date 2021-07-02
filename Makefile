TMPDIR := $(if $(TMPDIR),$(TMPDIR),"/tmp/")
GOPATH := $(shell go env GOPATH)

bin := $(GOPATH)/bin/tunk
gofiles := $(wildcard go.mod go.sum *.go **/*.go **/**/*.go **/**/**/*.go)

gocoverutil := $(GOPATH)/bin/gocoverutil
staticcheck := $(GOPATH)/bin/staticcheck
gomodoutdated := $(GOPATH)/bin/go-mod-outdated

all: build

build: $(bin)

$(bin): $(gofiles)
	GO111MODULE=on go install ./...

.PHONY: clean
clean:
	# git clean -x -f
	rm -rf $(TMPDIR)/tunk-*

.PHONY: test
test:
	GO111MODULE=on go test -short -cover ./...

.PHONY: test.race
test.race:
	GO111MODULE=on go test -race ./...

.PHONY: test.lint
test.lint: $(staticcheck)
	GO111MODULE=on $(staticcheck) -checks all ./...
	go vet ./...
	semgrep --error -c r/dgryski.semgrep-go -c p/gosec -c p/golang

.PHONY: test.cover
test.cover: SHELL:=/bin/bash
test.cover: $(gocoverutil)
	$(gocoverutil) -coverprofile=cov.out test -covermode=count ./... \
		2> >(grep -v "no packages being tested depend on matches for pattern" 1>&2) \
		| sed -e 's/of statements in .*/of statements/'
	@echo -n "total: "; go tool cover -func=cov.out | tail -n 1 | sed -e 's/\((statements)\|total:\)//g' | tr -s "[:space:]"

.PHONY: test.outdated
test.outdated: $(gomodoutdated)
	GO111MODULE=on go list -u -m -json all | go-mod-outdated -direct

.PHONY: release.dryrun
release.dryrun:
	goreleaser --snapshot --skip-publish --rm-dist

.PHONY: release
release:
	goreleaser --rm-dist

$(gocoverutil):
	GO111MODULE=off go get github.com/AlekSi/gocoverutil

$(staticcheck):
	cd $(TMPDIR) && GO111MODULE=on go get honnef.co/go/tools/cmd/staticcheck@2019.2.3

$(gomodoutdated):
	GO111MODULE=off go get github.com/psampaz/go-mod-outdated

