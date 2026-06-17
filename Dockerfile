FROM golang:1.26 AS build

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" -o /go/bin/app

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /go/bin/app /

USER nonroot:nonroot

ENTRYPOINT ["/app"]
