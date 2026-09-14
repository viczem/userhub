FROM golang:1.27.0-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY services/userhub-directory/go.mod services/userhub-directory/go.sum services/userhub-directory/
COPY services/userhub-directory services/userhub-directory

RUN CGO_ENABLED=0 go -C services/userhub-directory build -trimpath -ldflags="-s -w" -o /out/userhub-directory ./cmd

FROM scratch

USER 65532:65532
COPY --from=build /out/userhub-directory /userhub-directory
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/userhub-directory"]
CMD ["start"]
