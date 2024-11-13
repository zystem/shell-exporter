FROM alpine:3.20
RUN apk add --no-cache --update bash coreutils curl jq kubectl
COPY ./shell-exporter /bin/shell-exporter
COPY ./scripts/* /scripts/
EXPOSE 9000
CMD ["/bin/shell-exporter"]
