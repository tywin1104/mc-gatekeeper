SITEKEY?=6Ld7qcAUAAAAADtzQrmz7it_EOpygwqSKuC8nY5I
.PHONY: local
local:
	REACT_APP_RECAPTCHA_SITEKEY=$(SITEKEY) \
	REACT_APP_API_HOST=http://localhost:8080 \
	npm run start

.PHONY: local-docker
local-docker: ## Build docker image and run in localhost:80. Modify the nginx config to use localhost:8080 as upstream
	$(shell ./scripts/nginx_config_local.sh)
	docker build -t mc-whitelist-app:test .
	docker run -it -p 80:80 -e  \
	RECAPTCHA_SITEKEY=$(SITEKEY) \
	mc-whitelist-app:test

.PHONY: registry-image
registry-image: ## Builds docker image, uploads to registry, and sets dispatcher image on targeted cluster when necessary
	$(shell ./scripts/nginx_config_prod.sh)
	docker build -t mc-whitelist-app:$(shell git rev-parse --short HEAD) .
	docker tag mc-whitelist-app:$(shell git rev-parse --short HEAD) us.gcr.io/mc-whitelist-demo/mc-whitelist-app:$(shell git rev-parse --short HEAD)
	docker push us.gcr.io/mc-whitelist-demo/mc-whitelist-app:$(shell git rev-parse --short HEAD)
