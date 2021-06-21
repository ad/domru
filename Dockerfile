FROM golang:alpine as builder

ARG BUILD_ARCH

RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR $GOPATH/src/app/
COPY . .
COPY config.json /config.json
RUN mkdir -p /data && touch /data/account.json && chmod 777 /data/account.json

RUN CGO_ENABLED=0 go build -mod=vendor -ldflags='-w -s -extldflags "-static"' -a -o /go/bin/domru .

FROM scratch

ARG BUILD_DATE
ARG BUILD_REF

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /go/bin/domru /go/bin/domru
COPY --from=builder /config.json /config.json
COPY --from=builder /data/account.json /data/account.json

EXPOSE 18000

ENTRYPOINT ["/go/bin/domru"]

# Labels
LABEL \
    io.hass.name="Domofon addon" \
    io.hass.description="Domofon addon" \
    io.hass.arch="${BUILD_ARCH}" \
    io.hass.type="addon" \
    maintainer="ad <github@apatin.ru>" \
    org.label-schema.description="Domofon addon" \
    org.label-schema.build-date=${BUILD_DATE} \
    org.label-schema.name="Domofon addon" \
    org.label-schema.schema-version="1.0" \
    org.label-schema.usage="https://gitlab.com/ad/domru/-/blob/master/README.md" \
    org.label-schema.vcs-ref=${BUILD_REF} \
    org.label-schema.vcs-url="https://github.com/ad/domru/" \
    org.label-schema.vendor="HomeAssistant add-ons by ad"