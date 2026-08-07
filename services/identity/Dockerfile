FROM golang:1.26.5-alpine AS build

WORKDIR /src

COPY go.work ./
COPY services/iam/go.mod services/iam/go.mod
COPY services/identity/go.mod services/identity/go.mod
COPY shared/go.mod shared/go.sum shared/

RUN go -C shared mod download

COPY services/identity services/identity
COPY shared shared

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/identity ./services/identity/cmd

FROM scratch

USER 65532:65532
COPY --from=build /out/identity /identity

ENTRYPOINT ["/identity"]
CMD ["start"]
