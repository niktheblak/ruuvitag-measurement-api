FROM golang:1.24 AS build

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /go/bin/app

FROM ubuntu:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -q tzdata \
    && apt-get clean

COPY --from=build /go/bin/app /

ENTRYPOINT ["/app"]
