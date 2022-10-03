FROM golang:1.19
WORKDIR /go/src/Paktum
COPY . ./
RUN go build -o Paktum Paktum

FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN apk add gcompat
WORKDIR /root/
COPY --from=0 /go/src/Paktum/Paktum /root/Paktum
ENTRYPOINT ["/root/Paktum"]
