GOCMD=go
GOBUILD=$(GOCMD) build
GOGET=$(GOCMD) get
BIN=main
BUILD=$(shell pwd)/main

MAINLOC?=cmd/commitbuilder/main.go
CONTAINER?=generic-container
PORT?=9000

$(BUILD):
	$(GOBUILD) -o $(BIN) ${MAINLOC}

deps:
	$(GOGET) "github.com/random9s/cinder/..."
	$(GOGET) "gopkg.in/src-d/go-git.v4/..."
	$(GOGET) "github.com/gorilla/mux"

docker:
	docker build -t ${CONTAINER} -f Dockerfile .
	docker run -d --rm -p ${PORT}:8080 --name ${CONTAINER} ${CONTAINER}

clean:
	rm -rf log; echo > /dev/null
	rm $(BIN); echo > /dev/null
