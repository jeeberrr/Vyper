go install mvdan.cc/garble@latest
go mod tidy
go build -o builder.exe ./Builder/
