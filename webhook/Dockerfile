FROM golang:1.14-alpine as build
ENV GO111MODULE on
ENV CGO_ENABLED=0
WORKDIR /go/src/github.com/edgelesssys/marblerun/webhook
ADD . .
RUN go mod init
RUN go mod tidy
RUN go get -v ./...
#RUN go test -v ./...
RUN go build -v -o marblerun-webhook cmd/main.go

FROM scratch
WORKDIR /app
COPY --from=build /go/src/github.com/edgelesssys/marblerun/webhook/marblerun-webhook ./marblerun-webhook
ENTRYPOINT ["/app/marblerun-webhook"]