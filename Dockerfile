FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web .
RUN npm run build

FROM golang:1.21 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist internal/ui/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/agentary ./cmd/agentary

FROM gcr.io/distroless/static:nonroot

COPY --from=build /out/agentary /agentary

EXPOSE 3548

USER nonroot:nonroot

ENV AGENTARY_HOME=/data
# Default: run web + scheduler in foreground. Override with CMD for custom home.
ENTRYPOINT ["/agentary"]
CMD ["start", "--foreground", "--home", "/data"]

