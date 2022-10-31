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
RUN apk --no-cache add ca-certificates
RUN apk add gcompat ffmpeg
WORKDIR /root/
COPY --from=0 /go/src/Paktum/Paktum /root/Paktum
ENTRYPOINT ["/root/Paktum"]
