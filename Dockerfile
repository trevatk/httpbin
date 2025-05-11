
FROM oven/bun:1.2.12-alpine

WORKDIR /app

COPY . .

EXPOSE 8080
HEALTHCHECK NONE

CMD [ "bun", "index.ts" ]