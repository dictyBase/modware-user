FROM golang:1.11.5-alpine3.8
LABEL maintainer="Siddhartha Basu <siddhartha-basu@northwestern.edu>"
RUN apk add --no-cache git build-base
RUN mkdir -p /go/src/github.com/dictyBase/modware-user
WORKDIR /go/src/github.com/dictyBase/modware-user
COPY go.mod go.sum main.go ./
ADD server server
ADD commands commands
ADD message message
ADD validate validate
RUN go build -o app

FROM alpine:3.7
RUN apk --no-cache add ca-certificates
COPY --from=0 /go/src/github.com/dictyBase/modware-user/app /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/app"]
