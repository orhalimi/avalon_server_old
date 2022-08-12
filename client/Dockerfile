FROM node:16.16-alpine3.15 as BUILD
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm install
COPY . .
RUN npm run build --prod -- --output-path=./dist/out

FROM nginx:1.23.1-alpine
COPY --from=BUILD /app/dist/out/ /usr/share/nginx/html
COPY --from=BUILD /app/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80 12345
