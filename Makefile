linux:
	GOOS=linux go build -o go-url.linux

linux-docker:
	docker run --pull always --rm \
		-v gourlsvc-build-cache:/root/.cache/go-build \
		-v gourlsvc-pkg-mod:/go/pkg/mod \
		-v $(PWD)/:/go/src/cirello.io/gourlsvc \
		-w /go/src/cirello.io/gourlsvc golang:latest \
		/bin/bash -c 'make linux'
