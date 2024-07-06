# https://chemidy.medium.com/create-the-smallest-and-secured-golang-docker-image-based-on-scratch-4752223b7324
FROM golang:1.22.4-alpine as build

RUN apk update && \
    apk add --no-cache git ca-certificates && \
    update-ca-certificates

# Create appuser
ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR /opt/dnssync

COPY go.mod ./
COPY go.sum ./

RUN go mod download
RUN go mod verify

COPY . ./

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

RUN go build -ldflags="-w -s" -o ./dnssync ./cmd/dnssync/main.go

#########################
# Deploy
#########################
FROM alpine:latest

ENV INTERVAL "FILLME"
ENV DNS_RECORD "FILLME"
ENV CLOUDFLARE_ZONE_ID "FILLME"
ENV CLOUDFLARE_API_KEY "FILLME"

WORKDIR /opt

COPY --from=build /opt/dnssync/dnssync /opt/
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group

# Use an unprivileged user.
USER appuser:appuser

CMD ["/bin/sh", "-c", "/opt/dnssync --interval ${INTERVAL} --zone-id ${CLOUDFLARE_ZONE_ID} --dns-record ${DNS_RECORD} --api-key ${CLOUDFLARE_API_KEY}"]
