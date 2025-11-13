FROM node:lts-alpine AS builder

WORKDIR /app

COPY . .

# Install dory globally and build docs
RUN npm install -g @clidey/dory && dory build

CMD ["dory", "preview"]
