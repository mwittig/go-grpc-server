.PHONY: compileGRPC initProtoc

PROTOC_GEN_GO := $(shell go env GOPATH)/bin/
PROTOC := $(shell which protoc)

ifeq ($(PROTOC),)
	PROTOC = must-rebuild
endif

UNAME := $(shell uname)

$(PROTOC):
ifeq ($(UNAME), Darwin)
	brew install protobuf
endif
ifeq ($(UNAME), Linux)
	sudo apt-get install protobuf-compiler
endif

# Use as default in our context
GIT_HOST_URL ?= "git.mms-support.de"

initProtoc:
		go get -d google.golang.org/protobuf/cmd/protoc-gen-go@latest
		go get -d google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

orderService: initProtoc ./api/order-service.proto
		PATH="$(PATH):$(PROTOC_GEN_GO)" protoc --go_out=./internal/app/protos/ \
			--go-grpc_out=./internal/app/protos/ \
			--proto_path=./api/ \
			./api/order-service.proto


compileGRPC: orderService


generate:
	go generate ./...
	make compileGRPC

tidy:
	gci -w *
	gofmt -s -w .
	GOPRIVATE=$(GIT_HOST_URL) go mod tidy
