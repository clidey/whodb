FROM node:lts-alpine AS build-stage

WORKDIR /app

# Install pnpm and set it up globally in a safe, repeatable way
ENV PNPM_HOME="/root/.npm-global"
ENV PATH="${PNPM_HOME}/bin:${PATH}"
RUN npm install -g pnpm && \
    pnpm config set prefix "${PNPM_HOME}"

COPY . .

# Install dependencies and build docs ONLY if package.json exists
RUN if [ -f package.json ]; then \
      pnpm install --frozen-lockfile && \
      pnpm add -D @types/node && \
      pnpm add -g @clidey/dory && \
      dory build; \
    else \
      echo "Skipping pnpm install and dory build, no package.json found."; \
    fi

FROM nginx:alpine

RUN rm /etc/nginx/conf.d/default.conf && \
    rm -rf /usr/share/nginx/html/*

COPY --from=build-stage /app/dist /usr/share/nginx/html/docs

RUN cat > /etc/nginx/conf.d/docs.conf <<'EOF'
server {
    listen       80;
    listen  [::]:80;
    server_name  localhost;

    location / {
        root   /usr/share/nginx/html/docs;
        index  index.html index.htm;
        try_files $uri $uri/ /index.html;
    }

    error_page   500 502 503 504  /50x.html;
    location = /50x.html {
        root   /usr/share/nginx/html/docs;
    }
}
EOF
