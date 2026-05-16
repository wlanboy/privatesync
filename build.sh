env GOOS=linux GOARCH=amd64 go build -o privatesync-linux-amd64 .
env GOOS=windows GOARCH=amd64 go build -o privatesync-windows-amd64.exe .
