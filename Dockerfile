FROM debian:bullseye-slim as tools

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl wget git && \
    rm -rf /var/lib/apt/lists/*

FROM golang:latest

# Install git for git operations
# Install curl for coolify health check
# RUN apt-get install --no-install-recommends -y \
#     curl \
#     git \
#     && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives

# Create needed directories
RUN mkdir -p /tools/bin /tools/lib

# Copy wget & curl and their dynamic libraries
COPY --from=tools /usr/bin/curl /tools/bin/curl
COPY --from=tools /usr/bin/wget /tools/bin/wget
COPY --from=tools /usr/bin/git /tools/bin/git
# COPY --from=tools /lib /tools/lib
# COPY --from=tools /lib64 /tools/lib64
# COPY --from=tools /usr/lib /tools/usr-lib

# Optional: Add /tools/bin to PATH
ENV PATH="/tools/bin:$PATH"

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /server main.go

EXPOSE 3000

CMD ["/server"]