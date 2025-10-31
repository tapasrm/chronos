FROM oven/bun:1 AS builder

WORKDIR /app
COPY frontend/package.json frontend/bun.lockb ./
RUN bun install
COPY frontend/ .
RUN bun run build

# Final stage for NGINX
FROM nginx:1-alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
