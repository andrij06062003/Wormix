# wormix-api (Laravel). The source comes from the "api" build context (../wormix-api by default).
FROM composer:2 AS vendor
WORKDIR /app
COPY --from=api composer.json composer.lock ./
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist --no-interaction --ignore-platform-reqs
COPY --from=api . .
RUN rm -f .env database/database.sqlite \
 && composer dump-autoload --optimize --no-dev --no-scripts

FROM php:8.3-cli-alpine
WORKDIR /app
COPY --from=vendor /app /app
COPY docker/api-entrypoint.sh /usr/local/bin/api-entrypoint
RUN chmod +x /usr/local/bin/api-entrypoint \
 && mkdir -p /data storage/framework/cache storage/framework/sessions storage/framework/views storage/logs bootstrap/cache
ENV APP_ENV=production \
    APP_DEBUG=false \
    LOG_CHANNEL=stderr \
    DB_CONNECTION=sqlite \
    DB_DATABASE=/data/database.sqlite \
    PHP_CLI_SERVER_WORKERS=4
EXPOSE 8000
VOLUME ["/data"]
ENTRYPOINT ["api-entrypoint"]
