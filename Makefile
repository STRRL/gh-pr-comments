.PHONY: help build clean install

BINARY_NAME := gh-pr-comments

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  help     Show this help message"
	@echo "  build    Build the extension binary"
	@echo "  clean    Remove build artifacts"
	@echo "  install  Clean, build, and reinstall the extension"

build:
	go build -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)

install: clean build
	gh extension remove pr-comments || true
	gh extension install .
