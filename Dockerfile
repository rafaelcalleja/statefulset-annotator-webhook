FROM golang:1.18-alpine AS build

RUN apk add --update git
RUN apk add ca-certificates

WORKDIR /go/src/github.com/rafaelcalleja/statefulset-annotator-webhook

COPY . .

RUN go mod tidy && TAG=$(git describe --tags --abbrev=0) \
    && LDFLAGS=$(echo "-s -w -X main.version="$TAG) \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/webhook-mutate -ldflags "$LDFLAGS" cmd/main.go

# Building image with the binary
FROM scratch

COPY --from=build /go/bin/webhook-mutate /go/bin/webhook-mutate
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/go/bin/webhook-mutate"]
