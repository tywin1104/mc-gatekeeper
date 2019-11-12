cat <<EOF > nginx.extra.conf -
location /api/v1 {
	proxy_pass http://localhost:8080;
}
EOF