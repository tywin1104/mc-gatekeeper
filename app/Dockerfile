# Stage 0, "build-stage", based on Node.js, to build and compile the frontend
FROM tiangolo/node-frontend:10 as build-stage
WORKDIR /app
COPY package*.json /app/
RUN npm install
COPY ./ /app/
RUN npm run build
# Stage 1, based on Nginx, to have only the compiled app, ready for production with Nginx
FROM nginx:1.15
COPY --from=build-stage /app/build/ /usr/share/nginx/html
COPY scripts/docker-entrypoint.sh scripts/generate_config_js.sh /
RUN chmod +x docker-entrypoint.sh generate_config_js.sh
# Copy the default nginx.conf provided by tiangolo/node-frontend
COPY --from=build-stage /nginx.conf /etc/nginx/conf.d/default.conf
COPY ./nginx.extra.conf /etc/nginx/extra-conf.d/nginx.extra.conf
ENTRYPOINT ["/docker-entrypoint.sh"]