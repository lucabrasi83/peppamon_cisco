FROM golang:1.14.1-alpine as builder
COPY ./certs/rds-ca-2019-root.pem /usr/local/share/ca-certificates
RUN apk add --no-cache build-base git ca-certificates && update-ca-certificates 2>/dev/null || true
COPY . /go/src/github.com/lucabrasi83/peppamon_cisco
WORKDIR /go/src/github.com/lucabrasi83/peppamon_cisco
ENV GO111MODULE on
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -ldflags="-X github.com/lucabrasi83/peppamon_cisco/initializer.Commit=$(git rev-parse --short HEAD) \
    -X github.com/lucabrasi83/peppamon_cisco/initializer.Version=$(git describe --tags) \
    -X github.com/lucabrasi83/peppamon_cisco/initializer.BuiltAt=$(date +%FT%T%z) \
    -X github.com/lucabrasi83/peppamon_cisco/initializer.BuiltOn=$(hostname)" -o peppamon-cisco-collector


FROM scratch
LABEL maintainer="sebastien.pouplin@tatacommunications.com"
USER 1001
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/lucabrasi83/peppamon_cisco/banner.txt /
COPY --from=builder /go/src/github.com/lucabrasi83/peppamon_cisco/peppamon-cisco-collector /
CMD ["./peppamon-cisco-collector"]
