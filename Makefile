
VERSION := $(shell awk -F\" '/const Version = / {print $$2}' < main.go)

build:
	CGO_ENABLED=0 go build

test:
	go test -bench=. -benchmem

delivery:
	go clean
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
	tar cvfz cpubench1a-linux-x86_64-$(VERSION).tar.gz cpubench1a
	rm cpubench1a
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build
	tar cvfz cpubench1a-linux-Aarch64-$(VERSION).tar.gz cpubench1a
	rm cpubench1a
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build
	tar cvfz cpubench1a-darwin-Aarch64-$(VERSION).tar.gz cpubench1a
	rm cpubench1a

version:
	@echo '$(VERSION)'
