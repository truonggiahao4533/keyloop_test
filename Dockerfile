
#DEBIAN
#BUILD
FROM golang:1.25-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum main.go ./

RUN go mod download

#Copy necessary source code
COPY ./infrastructure ./infrastructure
COPY ./internal ./internal 
COPY ./migration ./migration

# RUN gf build and specify output path
RUN gf build -o ./bin/app


#RUNTIME
FROM debian:bookworm-slim

RUN apt-get update && \
  apt-get install -y libzmq5 \
  ca-certificates && \
  rm -rf /var/lib/apt/lists/*


WORKDIR /app
COPY --from=builder /src/bin/app .

RUN useradd -m appuser
USER appuser

ENTRYPOINT [ "/app/app" ]
