FROM golang:1.16-alpine AS builder
ENV CGO_ENABLED=0
ENV GOPRIVATE=""
ENV GOPROXY="https://goproxy.cn,direct"
ENV GOSUMDB="sum.golang.google.cn"
WORKDIR /root/remotecommand/

ADD . .
RUN go mod download \
    && go test --cover $(go list ./... | grep -v /vendor/) \
    && go build -o main cmd/server/main.go

FROM alpine

# The main mirrors are giving us timeout issues on builds periodically.
# So we can try these.
RUN sed -e 's/dl-cdn[.]alpinelinux.org/mirrors.aliyun.com/g' -i~ /etc/apk/repositories
RUN apk add --update --no-cache busybox-extras

# Setup Timezone
ENV TZ Asia/Shanghai
RUN apk add alpine-conf && \
    /sbin/setup-timezone -z Asia/Shanghai && \
    apk del alpine-conf

WORKDIR /root/

COPY --from=builder /root/remotecommand/main main
RUN chmod +x main

ENTRYPOINT ["/root/main"]
CMD []
