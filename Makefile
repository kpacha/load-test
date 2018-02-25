BIN_NAME :=load-test
DEP_VERSION=0.1.0
OS := $(shell uname | tr '[:upper:]' '[:lower:]')

all: deps build

prepare:
	@echo "Installing dep..."
	@curl -L -s https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-${OS}-amd64 -o ${GOPATH}/bin/dep
	@chmod a+x ${GOPATH}/bin/dep

deps:
	@echo "Setting up the vendors folder..."
	@dep ensure -v
	@echo ""
	@echo "Resolved dependencies:"
	@dep status
	@echo ""

build:
	@echo "Building the binary..."
	@go build -o ${BIN_NAME}
	@go install
	@echo "You can now use ./${BIN_NAME}"
