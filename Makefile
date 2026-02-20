BINARY_NAME=gtd
BUILD_DIR=bin

.PHONY: build clean install run test tidy

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME) 2>/dev/null || \
		cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

run: build
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

test:
	go test ./...

tidy:
	go mod tidy

init: build
	./$(BUILD_DIR)/$(BINARY_NAME) config init
