setup-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- -b $GOPATH/bin v1.24.0
setup:
	make setup-lint
lint:
	golangci-lint run
test:
	go test -race ./... -coverprofile=coverage.txt -covermode=atomic
coverage-html:
	go tool cover -html=coverage.txt -o coverage.html
test-all:
	make lint
	make test
