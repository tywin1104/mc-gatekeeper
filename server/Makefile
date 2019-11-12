.PHONY: registry-image
registry-image: ## Builds docker image, uploads to registry, and sets dispatcher image on targeted cluster when necessary
	docker build -t mc-whitelist-server:$(shell git rev-parse --short HEAD) .
	docker tag mc-whitelist-server:$(shell git rev-parse --short HEAD) us.gcr.io/mc-whitelist-demo/mc-whitelist-server:$(shell git rev-parse --short HEAD)
	docker push us.gcr.io/mc-whitelist-demo/mc-whitelist-server:$(shell git rev-parse --short HEAD)

