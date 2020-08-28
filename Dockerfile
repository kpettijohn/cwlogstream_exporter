FROM golang:1.14-alpine as builder

WORKDIR /src

ADD . .
ENV GO111MODULE=on GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o /opt/cwlogstream_exporter && \
    chmod 700 /opt/cwlogstream_exporter

FROM alpine

COPY --from=builder /opt/cwlogstream_exporter /opt/cwlogstream_exporter
EXPOSE 9520
