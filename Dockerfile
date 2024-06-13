FROM golang:1.22 as build

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /go/bin/app

FROM ubuntu:latest

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -q tzdata \
    && rm -r /var/cache/apt \
    && rm -r /var/lib/apt/lists

COPY --from=build /go/bin/app /

ENTRYPOINT ["/app", "server"]
