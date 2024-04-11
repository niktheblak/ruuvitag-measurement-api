FROM golang:1.22 as build

ENV DEBIAN_FRONTEND=noninteractive

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify
ADD . .
RUN go build -v -o /go/bin/app

FROM ubuntu:latest
RUN apt-get update && apt-get install -y tzdata
COPY --from=build /go/bin/app /
ENTRYPOINT ["/app", "server"]
