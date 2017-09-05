FROM alpine:3.6

COPY habitat-operator /habitat-operator

ENTRYPOINT ["/habitat-operator"]
