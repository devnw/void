FROM alpine:latest

RUN mkdir /etc/void
RUN apk --update add --no-cache ca-certificates

COPY void /bin/void
COPY config.yaml /etc/void/config.yaml

CMD /bin/void
