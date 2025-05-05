# project name
BINARY=gochat

# go command
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# compile parameters
LDFLAGS=-ldflags "-s -w"
DEBUG_LDFLAGS=-gcflags="all=-N -l"

BIN_DIR=bin

# default target
all: clean build

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# compile
build: $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) .

# debug mode compile
debug: $(BIN_DIR)
	$(GOBUILD) $(DEBUG_LDFLAGS) -o $(BIN_DIR)/$(BINARY) .

# clean
clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

# run test
test:
	$(GOTEST) -v ./...

# update dependencies
tidy:
	$(GOMOD) tidy

# generate proto files
proto:
	cd gateway/rpc && protoc -I service --go_out=service --go-grpc_out=service service/gateway.proto
	cd state/rpc && protoc -I service --go_out=service --go-grpc_out=service service/state.proto
	cd common/idl && protoc -I message  --go_out=message --go-grpc_out=message  message/message.proto

# help information
help:
	@echo "Available commands:"
	@echo " make all - Clean and build"
	@echo " make build - Build"
	@echo " make debug - Build for dlv debugging"
	@echo " make clean - Clean build files"
	@echo " make test - Run tests"
	@echo " make tidy - Update dependencies"
	@echo " make proto - Generate proto files"

.PHONY: all build debug clean test tidy proto help