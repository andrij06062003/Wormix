# One Wormix environment (prod or dev): its own Docker daemon (Docker-in-Docker) plus the deployer
# that keeps the Wormix stack in it on the head of one branch. See README.md.
FROM docker:29-dind

RUN apk add --no-cache bash curl git jq tar tzdata

COPY entrypoint.sh /usr/local/bin/sandbox-entrypoint.sh
COPY deployer.sh /usr/local/bin/wormix-deployer.sh
RUN chmod +x /usr/local/bin/sandbox-entrypoint.sh /usr/local/bin/wormix-deployer.sh

ENTRYPOINT ["sandbox-entrypoint.sh"]
