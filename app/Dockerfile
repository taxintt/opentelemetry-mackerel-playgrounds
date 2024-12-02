FROM golang:1.22 as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bootstrap

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/bootstrap /bootstrap
CMD ["/bootstrap"]