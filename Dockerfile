FROM golang:1.11.13-alpine3.10
LABEL maintainer="Siddhartha Basu <siddhartha-basu@northwestern.edu>"
ENV GOPROXY https://proxy.golang.org
RUN apk add --no-cache git build-base
RUN mkdir -p /modware-user
WORKDIR /modware-user
COPY go.mod ./
COPY go.sum ./
COPY main.go ./
RUN go mod download
ADD server server
ADD commands commands
ADD message message
ADD validate validate
RUN go build -o app

FROM alpine:3.10
RUN apk --no-cache add ca-certificates
COPY --from=0 /modware-user/app /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/app"]
