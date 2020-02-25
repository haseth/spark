ARG GOLANG_VERSION=1.13.4
FROM golang:${GOLANG_VERSION}

ENV GOPATH=/go:/src

WORKDIR /src

RUN go get \
    github.com/haseth/spark
    
COPY client.go /src/client.go

ENTRYPOINT ["go","run","/src/client.go"]
