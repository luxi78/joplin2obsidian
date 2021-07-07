VERSION=$(shell git describe --tags)

.PHONY: core 
core:
	@echo build version: $(VERSION)
	env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o dist/joplin2obsidian_darwin_amd64
	env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION)" -o dist/joplin2obsidian_darwin_arm64
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o dist/joplin2obsidian_linux_amd64
	env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o dist/joplin2obsidian_windows_amd64.exe