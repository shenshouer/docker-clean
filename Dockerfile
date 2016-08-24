FROM scratch

COPY docker-clean /docker-clean

ENTRYPOINT ["/docker-clean"]