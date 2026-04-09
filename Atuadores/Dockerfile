FROM golang:1.21-alpine

WORKDIR /app

COPY . .

RUN go mod init sensor && go mod tidy
RUN go build -o atuador-exec .

CMD ["./atuador-exec"]
