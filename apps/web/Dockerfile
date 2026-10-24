FROM node:24.22.1-alpine AS dependencies
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
FROM nginxinc/nginx-unprivileged:1.28.2-alpine AS runtime
COPY deploy/web/default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/apps/web/dist /usr/share/nginx/html
EXPOSE 8080
