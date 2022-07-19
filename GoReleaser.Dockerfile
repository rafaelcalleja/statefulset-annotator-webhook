FROM alpine:3.15.0
RUN apk add --update --no-cache ca-certificates
ENTRYPOINT ["/go/bin/webhook-mutate"]
COPY webhook-mutate /go/bin/webhook-mutate