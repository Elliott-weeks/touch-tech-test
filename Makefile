
deps:
	go mod download

test:
	go test ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

build:
	go build -o bin/main main.go

vet:
	go vet

