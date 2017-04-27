GO_EXECUTABLE ?= go
VERSION ?= $(shell git describe --tags --long --always --dirty)
OUTPUT_BIN = heraldbot

build:
	${GO_EXECUTABLE} build -o ${OUTPUT_BIN} -ldflags "-X main.version=${VERSION}"

run: build
	./${OUTPUT_BIN} --token 'MzA2MTg1ODk0NjkwMjkxNzE0.C-AZyA.JzWpgczE7vumYuwE_6f_5Is4MDo' --channel-listen WWP:general --channel-announce WWP:general --admin alinsa#9054 --admin mcahogarth#8422
