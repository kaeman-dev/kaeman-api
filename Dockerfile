FROM --platform=$BUILDPLATFORM alpine AS certs
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/kaeman-public-api /app/kaeman-public-api
WORKDIR /app
ENTRYPOINT ["/app/kaeman-public-api"]
