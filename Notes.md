# cross build from macos
brew install mingw-w64
https://mt165.co.uk/blog/static-link-go/

GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build main.go
GOOS=windows GOARCH=amd64  go build -o kindatest main.go

ALSO:
https://go.dev/wiki/WindowsDLLs