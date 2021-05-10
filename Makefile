export GO_BUILD=env GO111MODULE=on go build
export GO_TEST=env GO111MODULE=on go test
default: build
build:
	$(GO_BUILD)
