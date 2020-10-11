# privatesync
privatesync

# build
go get -d -v

go clean

go build

# run
go run main.go

# debug
go get -u github.com/go-delve/delve/cmd/dlv
dlv debug ./goservice
