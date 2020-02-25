ARG GOLANG_VERSION=1.13.4
FROM golang:${GOLANG_VERSION}

ENV GOPATH=/go:/src

WORKDIR /src

RUN go get \
    github.com/haseth/spark
    
COPY server.go /src/server.go
EXPOSE 8080
ENTRYPOINT ["go","run","/src/server.go"]
