FROM node:18-alpine
RUN mkdir -p /home/node/app/node_modules && chown -R node:node /home/node/app && chmod -R 777 /home/node/app
WORKDIR /home/node/app
RUN npm install -g pnpm
COPY paktum-fe/package*.json ./
COPY paktum-fe/pnpm-lock.yaml ./
USER node
RUN pnpm install
COPY --chown=node:node paktum-fe .
RUN NODE_ENV=production npm run build

FROM golang:1.19
ARG COMMIT_REF
ENV COMMIT_REF=$COMMIT_REF

WORKDIR /go/src/Paktum

# Install deps
COPY go.mod go.sum ./
RUN go mod download -x

# Generate GraphQL schema
COPY graph ./graph
COPY tools.go gqlgen.yml ./
RUN go run github.com/99designs/gqlgen generate

# Copy over built frontend
COPY --from=0 /home/node/app/dist ./paktum-fe/dist

# Build code
COPY . ./
COPY .gi[t] ./.git
RUN go generate ./...
RUN go build -v -o Paktum Paktum

FROM alpine:latest
LABEL maintainer="privateger@privateger.me"

# Install updates & packages
RUN apk --no-cache add ca-certificates && apk add gcompat ffmpeg

# Create Paktum user
RUN addgroup -S paktum && adduser -S paktum -G paktum
USER paktum
WORKDIR /home/paktum

# Create images volume directory mountpoint
RUN mkdir -p /home/paktum/images
VOLUME /home/paktum/images

# Copy over Paktum
COPY --from=1 /go/src/Paktum/Paktum /home/paktum

EXPOSE 8080

ENTRYPOINT ["/home/paktum/Paktum"]
