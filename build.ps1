$gitHash = (git rev-list -1 HEAD 2>&1)
go build -ldflags "-X main.GitHash=${gitHash}"
