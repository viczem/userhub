FROM golang:1.27.0-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.work ./
COPY services/userhub/go.mod services/userhub/go.sum services/userhub/

RUN go -C services/userhub mod download

COPY services/userhub services/userhub

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/userhub ./services/userhub/cmd

FROM scratch

USER 65532:65532
COPY --from=build /out/userhub /userhub
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/userhub"]
CMD ["start"]
