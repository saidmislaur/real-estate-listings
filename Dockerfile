FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /flatly ./cmd/app

FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=builder /flatly /app/flatly

EXPOSE 8082

ENTRYPOINT ["/app/flatly"]
