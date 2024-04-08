FROM golang:1.20

WORKDIR /app

COPY go.mod .
COPY main.go .

RUN go mod tidy
RUN go build -o bin .

ENTRYPOINT [ "/app/bin" ]