linux:
	GOOS=linux go build -o go-url.linux

linux-docker:
	docker run --pull always -ti --rm -v $(PWD)/:/go/src/cirello.io/gourlsvc \
		-w /go/src/cirello.io/gourlsvc golang:latest \
		/bin/bash -c 'make linux'
