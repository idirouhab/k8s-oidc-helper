FROM golang:1.9-alpine as builder

LABEL maintainer="ouhab.idir@gmail.com"

ENV PKG=/go/src/github.com/idirouhab/k8s-oidc-helper
COPY . $PKG
WORKDIR $PKG

RUN go install -ldflags '-w'

FROM alpine:latest

RUN apk update
RUN apk add ca-certificates

COPY --from=builder /go/bin/k8s-oidc-helper /bin/k8s-oidc-helper

ENTRYPOINT ["/bin/k8s-oidc-helper"]
