FROM golang:1.9 as builder

RUN mkdir -p $GOPATH/src/github.com/Visteras/sharex-uploader
COPY ./ $GOPATH/src/github.com/Visteras/sharex-uploader
WORKDIR $GOPATH/src/github.com/Visteras/sharex-uploader

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sharex-uploader .

FROM alpine

RUN mkdir -p /app/files

COPY --from=builder /go/src/github.com/Visteras/sharex-uploader /app/sharex-uploader
RUN chmod +x /app/sharex-uploader

WORKDIR /app

CMD ["/app/sharex-uploader"]