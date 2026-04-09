FROM golang:1.21-alpine

WORKDIR /app

COPY . .

RUN go mod init cliente || true
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o cliente-exec .

RUN chmod +x cliente-exec

CMD ["./cliente-exec"]
