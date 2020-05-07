test:
	go test -race ./... -coverprofile=coverage.txt -covermode=atomic
coverage-html:
	go tool cover -html=coverage.txt -o coverage.html
test-all:
	make test
	make coverage-html
db-create-migration:
	migrate create -ext sql -dir db/mysql $(NAME)
db-migrate:
	migrate -database mysql://root:rootpw@/letter_block_development -source file://db/mysql up
db-rollback:
	migrate -database mysql://root:rootpw@/letter_block_development -source file://db/mysql down
