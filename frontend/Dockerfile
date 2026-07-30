FROM node:22-bookworm-slim AS development
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
EXPOSE 3000
CMD ["npm", "run", "dev", "--", "--hostname", "0.0.0.0"]
