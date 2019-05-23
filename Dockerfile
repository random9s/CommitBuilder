FROM ubuntu:18.04
LABEL maintainer="Jake Parham <jparham@fyusion.com>"

## Install dependencies
RUN apt-get update && apt-get -y upgrade && apt-get install -y wget git && apt-get clean
WORKDIR /tmp
RUN wget https://dl.google.com/go/go1.11.linux-amd64.tar.gz && tar -xvf go1.11.linux-amd64.tar.gz
RUN mv go /usr/local/

## setup go env
RUN mkdir -p /go/{src,pkg,bin}
ENV GOROOT="/usr/local/go"
ENV GOPATH="/go"
ENV GOBIN="/go/bin"
ENV PATH="${GOPATH}/bin:${GOROOT}/bin:${PATH}"

## Install Go dependencies
RUN go get "github.com/random9s/cinder/..."
RUN go get "gopkg.in/src-d/go-git.v4/..."
RUN go get "github.com/gorilla/mux"

### Add project source code
RUN mkdir -p /go/src/github.com/random9s/CommitBuilder/
WORKDIR /go/src/github.com/random9s/CommitBuilder/
COPY cmd cmd 
COPY pkg pkg
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -v -o main cmd/commitbuilder/main.go

EXPOSE 8080
ENTRYPOINT exec ./main > /var/log/server.log 2>&1
