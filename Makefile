default:
	go build -ldflags="-s -w" .

static:
	CGO_ENABLED=0 go build -ldflags="-s -w" .

staticARM:
	CGO_ENABLED=0 GOARCH=arm go build -ldflags="-s -w" .

run: default
	sudo ./simpleflash
