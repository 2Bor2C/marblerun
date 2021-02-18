FROM golang:1.14-alpine as build
ENV GO111MODULE on
ENV CGO_ENABLED=0
WORKDIR /go/src/github.com/edgelesssys/marblerun/webhook
ADD . .
RUN go build -v -o marblerun-webhook cmd/main.go

FROM scratch
WORKDIR /app
COPY --from=build /go/src/github.com/edgelesssys/marblerun/webhook/marblerun-webhook ./marblerun-webhook
ENTRYPOINT ["/app/marblerun-webhook"]
