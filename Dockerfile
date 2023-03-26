# Build stage.
FROM golang:1.20-alpine as builder

# Add git tool to extract latest commit and tag.
# Add root certificates to be used for ssl/tls.
# Add openssl to build self-signed certificates.
RUN apk add --update --no-cache ca-certificates git openssl

# Setup the working directory
WORKDIR /app/

# Copy go mod file and download dependencies.
COPY go.* ./
RUN go mod download -x

# Copy all files to the container’s workspace.
COPY . .

# Build the app program inside the container.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app.demo.redis -a -ldflags "-extldflags '-static' -X 'main.GitCommit=$(git rev-parse --short HEAD)' -X 'main.GitTag=$(git describe --tags --abbrev=0)' -X 'main.BuildTime=$(date -u '+%Y-%m-%d %I:%M:%S %p GMT')'" .

# Final stage with minimalist image.
FROM scratch

LABEL maintainer="Jerome Amon <https://blog.cloudmentor-scale.com>"

# Copy the static executable to the new container root.
COPY --from=builder ./app/app.demo.redis ./app.demo.redis

# Copy all ca-certs files to the same path.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 8080
ENTRYPOINT [ "./app.demo.redis" ]
