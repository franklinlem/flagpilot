FROM node:24.22.1-alpine@sha256:06d7adf3e0ef61a13f1d28682726366f97a15aa1888c934d3ecec86c33918b53 AS dependencies
WORKDIR /app
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
COPY packages/sdk-js/package.json packages/sdk-js/package.json
RUN npm ci
FROM dependencies AS development
COPY apps/web/ apps/web/
RUN chown -R node:node /app/apps/web
USER node
EXPOSE 5173
CMD ["npm", "run", "dev", "--workspace", "@flagpilot/web", "--", "--host", "0.0.0.0"]
FROM dependencies AS build
COPY apps/web/ apps/web/
RUN npm run build --workspace @flagpilot/web
FROM nginx/nginx-unprivileged:1.30.6-alpine@sha256:517607ae849a46afbd400616a783e4ead0290f52d20ccc0618ccea1a49362474 AS runtime
USER root
RUN apk add --no-cache --upgrade brotli-libs=1.2.1-r0 musl=1.2.6-r3 musl-utils=1.2.6-r3 zlib=1.3.2-r1 curl=8.23.0-r0 libcurl=8.23.0-r0
USER 101
COPY deploy/web/default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/apps/web/dist /usr/share/nginx/html
EXPOSE 8080
