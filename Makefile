test:
	go test -race ./... -coverprofile=coverage.txt -covermode=atomic
coverage-html:
	go tool cover -html=coverage.txt -o coverage.html
test-all:
	make test
	make coverage-html
