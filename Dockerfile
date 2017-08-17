FROM alpine:3.6

COPY operator /operator

ENTRYPOINT ["/operator"]
