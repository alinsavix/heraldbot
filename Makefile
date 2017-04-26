GO_EXECUTABLE ?= go
VERSION ?= $(shell git describe --tags --long --always --dirty)
OUTPUT_BIN = heraldbot

build:
	${GO_EXECUTABLE} build -o ${OUTPUT_BIN} -ldflags "-X main.version=${VERSION}"

run: build
	./${OUTPUT_BIN}
