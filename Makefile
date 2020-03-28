setup:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $GOPATH/bin v1.24.0
lint:
	golangci-lint run
test:
	go test -race -coverprofile=coverage.txt -covermode=atomic
test-all:
	make lint
	make test