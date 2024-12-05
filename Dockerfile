FROM golang:1.19
WORKDIR /go/src
COPY go ./go
COPY main.go .
COPY go.sum .
COPY go.mod .
ENV CGO_ENABLED=0
RUN go build -o openapi .
ENV GIN_MODE=release
EXPOSE 8080/tcp
ENTRYPOINT ["./openapi"]