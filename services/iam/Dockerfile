FROM golang:1.27.0-alpine AS build

WORKDIR /src

COPY go.work ./
COPY services/iam/go.mod services/iam/go.mod
COPY services/identity/go.mod services/identity/go.mod
COPY shared/go.mod shared/go.sum shared/

RUN go -C shared mod download

COPY services/iam services/iam
COPY shared shared

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/iam ./services/iam/cmd

FROM scratch

USER 65532:65532
COPY --from=build /out/iam /iam

ENTRYPOINT ["/iam"]
CMD ["start"]
