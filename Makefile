
VERSION := $(shell awk -F\" '/const Version = / {print $$2}' < main.go)

build:
	CGO_ENABLED=0 go build

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o cpubench1a.exe

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
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o cpubench1a.exe
	zip cpubench1a-windows-x86_64-$(VERSION).zip cpubench1a.exe
	rm cpubench1a.exe
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o cpubench1a.exe
	zip cpubench1a-windows-Aarch64-$(VERSION).zip cpubench1a.exe
	rm cpubench1a.exe

version:
	@echo '$(VERSION)'
