FROM docker:dind

RUN apk add --no-cache bash curl git jq

COPY entrypoint.sh /usr/local/bin/sandbox-entrypoint.sh
RUN chmod +x /usr/local/bin/sandbox-entrypoint.sh

WORKDIR /workspace

ENTRYPOINT ["sandbox-entrypoint.sh"]
