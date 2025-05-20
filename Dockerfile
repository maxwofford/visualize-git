FROM debian:bullseye-slim AS tools

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl wget git && \
    rm -rf /var/lib/apt/lists/*

FROM golang:latest

RUN mkdir -p /tools/bin /tools/lib
COPY --from=tools /usr/bin/curl /usr/bin/wget /usr/bin/git /tools/bin/
ENV PATH="/tools/bin:$PATH"

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /server main.go

EXPOSE 3000

CMD ["/server"]