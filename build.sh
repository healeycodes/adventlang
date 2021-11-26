GOOS=linux GOARCH=amd64 go build -o adventlang-linux cmd/advent-lang.go
GOOS=windows GOARCH=amd64 go build -o adventlang-windows cmd/adventlang.go
GOOS=darwin GOARCH=amd64 go build -o adventlang-darwin cmd/adventlang.go
GOOS=darwin GOARCH=arm64 go build -o adventlang-darwin-arm64 cmd/adventlang.go