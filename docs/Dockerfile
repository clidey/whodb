FROM node:lts-alpine AS builder

WORKDIR /app

COPY . .

# Install dory globally and build docs
RUN npm install -g @clidey/dory && dory build

FROM nginx:alpine

RUN rm /etc/nginx/conf.d/default.conf && \
    rm -rf /usr/share/nginx/html/*

COPY --from=builder /app/dist /usr/share/nginx/html/docs

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

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
