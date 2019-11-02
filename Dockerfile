FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git
WORKDIR $GOPATH/src/github.com/tehmoon/cryptocli
COPY src/cryptocli .
RUN go get -d -v
ARG VERSION
RUN go build -tags netgo -o /go/bin/cryptocli -ldflags "-X 'main.VERSION=${VERSION}' -extldflags \"-static\" -s -w"

FROM scratch
COPY --from=builder /go/bin/cryptocli /go/bin/cryptocli
ENTRYPOINT ["/go/bin/cryptocli"]
