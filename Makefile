GO_EXECUTABLE ?= go
VERSION ?= $(shell git describe --tags --long --always --dirty)
OUTPUT_BIN = heraldbot
GOPATH ?= ${HOME}/go
TOKEN ?= replace_me_with_real_token

build:
	GOPATH=${GOPATH} ${GO_EXECUTABLE} build -o ${OUTPUT_BIN} -ldflags "-X main.version=${VERSION}"

run: build
	./${OUTPUT_BIN} --token '${TOKEN}' --channel-listen WWP:general --channel-announce WWP:general --admin alinsa#9054 --admin mcahogarth#8422
