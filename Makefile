.PHONY: build install clean test

BINARY=gitty
INSTALL_PATH=$(HOME)/.local/bin

build:
	go build -o $(BINARY) .

install: build
	mkdir -p $(INSTALL_PATH)
	cp $(BINARY) $(INSTALL_PATH)/
	@echo "Installed $(BINARY) to $(INSTALL_PATH)"
	@echo "Make sure $(INSTALL_PATH) is in your PATH"

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY)
	@echo "Uninstalled $(BINARY)"

clean:
	rm -f $(BINARY)
	go clean

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

# Create default config
config:
	mkdir -p $(HOME)/.config/gitty
	@if [ ! -f $(HOME)/.config/gitty/config.yaml ]; then \
		cp config.example.yaml $(HOME)/.config/gitty/config.yaml; \
		echo "Created config at $(HOME)/.config/gitty/config.yaml"; \
	else \
		echo "Config already exists at $(HOME)/.config/gitty/config.yaml"; \
	fi
