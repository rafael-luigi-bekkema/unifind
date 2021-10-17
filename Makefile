GOARCH=$(shell go env GOARCH)
GOOS=$(shell go env GOOS)
BIN_NAME := unifind
BIN := build/${GOARCH}/${GOOS}/${BIN_NAME}
DEPS := $(shell find . -iname '*.go') go.mod
DESTDIR := ${HOME}/.local/bin
CGO_ENABLED := 0

${BIN}: ${DEPS}
	CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${BIN}

.PHONY: install	
install: ${BIN}
	install -m 755 -d ${DESTDIR}
	install -m 755 -Cv ${BIN} ${DESTDIR}

.PHONY: clean
clean:
	! test -e ${BIN} || rm ${BIN}
	! test -e build || rmdir build

