FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/enforcer ./cmd/enforcer

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/enforcer /enforcer
COPY policies/example.yaml /policies/example.yaml

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/enforcer", "-policy-file=/policies/example.yaml"]
