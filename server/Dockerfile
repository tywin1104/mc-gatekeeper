FROM golang:1.12.0 AS builder

LABEL maintainer="tiaven1104@gmail.com"
RUN mkdir /server
ADD . /server

WORKDIR /server/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o mc-whitelist-server

FROM alpine:latest AS production
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN apk add --no-cache bash
COPY --from=builder /server/cmd/mc-whitelist-server /server/mc-whitelist-server
COPY --from=builder /server/mailer/templates /server/mailer/templates/
WORKDIR /server
CMD ["./mc-whitelist-server"]
