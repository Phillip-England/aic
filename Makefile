docs:
	bun ./www/**/*.html

dev:
	air

test:
	clear; go test ./...
