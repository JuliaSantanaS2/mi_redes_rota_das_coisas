FROM golang:1.21-alpine

WORKDIR /app

COPY . .

# Inicializa o modulo (caso não tenha o go.mod criado) e baixa dependencias
RUN go mod init broker || true
RUN go mod tidy

RUN go build -o broker-exec .

CMD ["./broker-exec"]
