FROM golang:1.21-alpine

WORKDIR /app

COPY . .

RUN go mod init sensor && go mod tidy
RUN go build -o sensor-exec .

CMD ["./sensor-exec"]
