FROM alpine:3.7

RUN apk add --no-cache ca-certificates

ADD node-group-controller-manager /bin/

CMD ["/bin/node-group-controller-manager"]