dev:
	go run ./cmd/aic/main.go

build:
	go build -o ./tmp/main ./cmd/aic

kill:
	lsof -ti:8000 | xargs kill -9 || true