FROM node:22-bookworm-slim

WORKDIR /app

# Nuxt écoute sur 0.0.0.0:8080 (port publié par docker-compose.yml)
ENV NUXT_HOST=0.0.0.0 NUXT_PORT=8080

CMD ["/bin/bash", "-c", "npm install && npm run dev"]
