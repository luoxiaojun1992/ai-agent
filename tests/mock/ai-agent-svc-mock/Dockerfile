FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/mock-ai-agent-svc .

FROM alpine:3.22
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/mock-ai-agent-svc /app/mock-ai-agent-svc
USER app
EXPOSE 18080
ENTRYPOINT ["/app/mock-ai-agent-svc"]
