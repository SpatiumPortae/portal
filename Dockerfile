# Mutli-stage build.
FROM golang:1.18-alpine3.14 as build-stage

# Copy source code and build binary
RUN mkdir /usr/app
COPY . /usr/app
WORKDIR /usr/app
RUN go build  -o app ./cmd/portal/*.go

# Copy binary from build container and build image.
FROM alpine:3.14
RUN mkdir /usr/app 
WORKDIR /usr/app
COPY --from=build-stage /usr/app/app .

ENTRYPOINT [ "./app", "serve","-p", "8080" ]
