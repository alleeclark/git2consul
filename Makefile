PACKAGE_VERSION ?= $(shell git describe --always --tags)
REGISTRY ?= dockerhub.com/alleeclark/git2consul
ARGS ?= --consul-addr="172.17.0.1:8500" --git-url="https://github.com/alleeclark/test-git2consul.git" --git-branch="origin" sync

.PHONY: images
images:
	docker build -t git2consul:$(PACKAGE_VERSION) -f docker/Dockerfile ./

.PHONY: publish
publish: images
	docker push $(REGISTRY)/git2consul:$(PACKAGE_VERSION)

.PHONY: devcluster
devcluster:
	docker run -itd --name consul --network=host consul:latest
	docker run -it git2consul:$(shell git describe --always --tags) $(ARGS)

.PHONY: refreshconsul
refreshconsul:
	docker restart consul