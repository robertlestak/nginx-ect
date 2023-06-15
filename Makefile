VERSION=v0.0.1

bin: bin/nginx-ect_darwin bin/nginx-ect_linux 

bin/nginx-ect_darwin:
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/nginx-ect_darwin cmd/nginx-ect/*.go
	openssl sha512 bin/nginx-ect_darwin > bin/nginx-ect_darwin.sha512

bin/nginx-ect_linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/nginx-ect_linux cmd/nginx-ect/*.go
	openssl sha512 bin/nginx-ect_linux > bin/nginx-ect_linux.sha512