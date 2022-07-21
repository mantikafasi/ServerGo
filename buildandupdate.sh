rm .\\server-go
GOOS=linux GOARCH=arm go build
scp .\\server-go oracle:/home/manti/server-go/