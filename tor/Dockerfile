FROM alpine:latest
RUN apk add --no-cache tor

ARG USER_ID=1000
ARG GROUP_ID=1000
# Reuse alpine's tor user but remap it to the host uid/gid
RUN deluser tor 2>/dev/null || true; \
    addgroup -g ${GROUP_ID} tor && \
    adduser -D -u ${USER_ID} -G tor tor

COPY tor/torrc /etc/tor/torrc
COPY tor/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 9050
ENTRYPOINT ["/entrypoint.sh"]
