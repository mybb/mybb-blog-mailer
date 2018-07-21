BINARY = ./bin/mybb-blog-mailer
GOARCH = amd64

all: clean linux darwin windows

clean:
	-rm -f ${BINARY}-*

linux: 
	GOOS=linux GOARCH=${GOARCH} go build -o ${BINARY}-linux-${GOARCH} .

darwin:
	GOOS=darwin GOARCH=${GOARCH} go build -o ${BINARY}-darwin-${GOARCH} .

windows:
	GOOS=windows GOARCH=${GOARCH} go build -o ${BINARY}-windows-${GOARCH}.exe .

.PHONY: clean linux darwin windows