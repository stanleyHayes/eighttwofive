FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bin/server /server
EXPOSE 8080
# The binary probes its own /healthz in -healthcheck mode, so this works on a
# distroless image with no shell or curl.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/server", "-healthcheck"]
ENTRYPOINT ["/server"]
