FROM golang:1.19
LABEL maintainer="privateger@privateger.me"

WORKDIR /go/src/Paktum

# Install deps
COPY go.mod go.sum ./
RUN go mod download -x

# Generate GraphQL schema
COPY graph ./graph
COPY tools.go gqlgen.yml ./
RUN go run github.com/99designs/gqlgen generate

# Build code
COPY . ./
RUN go build -o Paktum Paktum

FROM alpine:latest

# Install updates & packages
RUN apk --no-cache add ca-certificates
RUN apk add gcompat ffmpeg

# Create Paktum user
RUN addgroup -S paktum && adduser -S paktum -G paktum
USER paktum
WORKDIR /home/paktum

# Copy over Paktum and create images dir
COPY --from=0 /go/src/Paktum/Paktum /home/paktum
RUN mkdir -p /home/paktum/images
VOLUME /home/paktum/images

ENTRYPOINT ["/home/paktum/Paktum"]
