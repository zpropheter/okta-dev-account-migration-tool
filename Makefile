all: build

build:
	go build ./...

cli:
	cd vendor/github.com/okta/okta-sdk-golang/v5/okta/ && go install .

install: cli build
	go install ./...
