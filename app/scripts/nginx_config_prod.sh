cat <<EOF > nginx.extra.conf -
location /api/v1 {
  proxy_pass http://service-mc-whitelist-backend;
}
EOF