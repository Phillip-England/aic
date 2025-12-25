dev:
	go run main.go

kill:
	lsof -ti:8000 | xargs kill -9 || true
