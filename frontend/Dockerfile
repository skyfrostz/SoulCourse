# Production Dockerfile — Multi-stage build for Vue 3 + Vite frontend

FROM node:24-alpine AS build

WORKDIR /app

# Install pnpm
RUN corepack enable && corepack prepare pnpm@11.7.0 --activate

# Copy dependency manifests
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source code and build
COPY . .
RUN pnpm build

# --- Runtime stage ---
FROM nginx:1.29-alpine AS runtime

# Copy built files
COPY --from=build /app/dist /usr/share/nginx/html

# Copy nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
