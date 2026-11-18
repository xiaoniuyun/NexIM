# 构建阶段
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm install --no-audit --no-fund
COPY . .
RUN npm run build

# 运行阶段：由 Nginx 托管静态产物
FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html/h5
COPY nginx-h5.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
