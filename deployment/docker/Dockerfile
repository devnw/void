FROM alpine:latest

RUN mkdir /etc/void

COPY void /bin/void
RUN chmod +x /bin/void

COPY deployment/config/config.yaml /etc/void/config.yaml

ENTRYPOINT [ "/bin/void" ]
