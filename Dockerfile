
#DEBIAN
#BUILD
FROM golang:1.25-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum cmd/api/main.go ./

RUN go mod download

#Copy necessary source code
COPY ./infrastructure ./infrastructure
COPY ./internal ./internal 
COPY ./migration ./migration
COPY ./docs ./docs

# RUN go build and specify output path
RUN go build -o ./bin/app


#RUNTIME
FROM debian:bookworm-slim

RUN apt-get update 


WORKDIR /app
COPY --from=builder /src/bin/app .

RUN useradd -m appuser
USER appuser

ENTRYPOINT [ "/app/app" ]
