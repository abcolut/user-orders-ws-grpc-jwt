build:
	go build -o bin/server ./cmd/server
	go build -o bin/client ./cmd/client

run-server:
	go run ./cmd/server/main.go

run-client:
	go run ./cmd/client/main.go

clean:
	rm -rf bin/

test:
	go test ./...
