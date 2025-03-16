FROM golang:latest

# Install git for git operations
# Install curl for coolify health check
RUN apt-get install --no-install-recommends -y \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /server main.go

EXPOSE 3000

CMD ["/server"]