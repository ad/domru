FROM golang:alpine as builder
RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates

ENV USER=appuser
ENV UID=10001

RUN adduser --disabled-password --gecos "" --home "/nonexistent" --shell "/sbin/nologin" --no-create-home --uid "${UID}" "${USER}"
WORKDIR $GOPATH/src/app/
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags='-w -s -extldflags "-static"' -a -o /go/bin/domru .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /go/bin/domru /go/bin/domru

USER appuser:appuser

EXPOSE 8080

ENTRYPOINT ["/go/bin/domru"]