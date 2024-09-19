default: install

run: install
	gtl

install:
	go install ./cmd/...

check:
	staticcheck ./...

test:
	go test ./...
