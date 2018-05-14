FROM scratch

COPY habitat-operator /habitat-operator

ENTRYPOINT ["/habitat-operator", "-logtostderr"]
