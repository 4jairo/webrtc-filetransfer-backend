FROM chainguard/go as builder

WORKDIR /usr/src/webrtc-filetransfer

COPY . .

RUN go mod download

RUN go build -o /usr/local/bin/webrtc-filetransfer

FROM ubuntu:22.04

COPY --from=builder /usr/local/bin/webrtc-filetransfer /usr/local/bin/webrtc-filetransfer

EXPOSE 8900

CMD ["webrtc-filetransfer"]