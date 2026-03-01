from node:25.7.0-alpine3.22 AS builder

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

    # Special CORS headers for embed.js and embed-widget.js only
    location ~ ^/(embed|embed-widget)\.js$ {
        root   /usr/share/nginx/html/docs;
        try_files $uri $uri/ /$1.js =404;

        add_header 'Access-Control-Allow-Origin' '*' always;
        add_header 'Access-Control-Allow-Methods' 'GET' always;
        add_header 'Access-Control-Allow-Headers' '*' always;

        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' '*' always;
            add_header 'Access-Control-Allow-Methods' 'GET' always;
            add_header 'Access-Control-Allow-Headers' '*' always;
            add_header 'Content-Length' 0;
            add_header 'Content-Type' 'text/plain; charset=UTF-8';
            return 204;
        }
    }

    # CORS headers for images
    location ~ ^/images/.*\.(png|jpg|jpeg|gif|svg|webp|ico)$ {
        root   /usr/share/nginx/html/docs;
        try_files $uri =404;

        add_header 'Access-Control-Allow-Origin' '*' always;
        add_header 'Access-Control-Allow-Methods' 'GET' always;
        add_header 'Access-Control-Allow-Headers' '*' always;
        add_header 'Cache-Control' 'public, max-age=31536000' always;

        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' '*' always;
            add_header 'Access-Control-Allow-Methods' 'GET' always;
            add_header 'Access-Control-Allow-Headers' '*' always;
            add_header 'Content-Length' 0;
            add_header 'Content-Type' 'text/plain; charset=UTF-8';
            return 204;
        }
    }

    # All other static files and docs site
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
