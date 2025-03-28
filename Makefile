BIN_NAME :=load-test

all: build

build:
	@echo "Building the binary..."
	@go build -o ${BIN_NAME}
	@go install
	@echo "You can now use ./${BIN_NAME}"
