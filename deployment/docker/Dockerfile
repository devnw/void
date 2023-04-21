FROM alpine:latest

RUN mkdir /etc/void
RUN apk --update add --no-cache ca-certificates

COPY void /bin/void
COPY deployment/config/config.yaml /etc/void/config.yaml

CMD /bin/void
