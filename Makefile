all:
	go build

noip-armv6:
	GOARCH=arm GOARM=6 go build -o noip-armv6

noip-armv7:
	GOARCH=arm GOARM=7 go build -o noip-armv7
