# syntax=docker/dockerfile:1
# docker build --pull --tag avalon-server -f ".\server.Dockerfile" .

FROM golang:1.16-alpine
WORKDIR /app
COPY ./avalon_server_old/go.mod ./
COPY ./avalon_server_old/go.sum ./
RUN go mod download
COPY  ./avalon_server_old/*.go ./
RUN go build -o bin/main
EXPOSE 12345
CMD ["./bin/main"]