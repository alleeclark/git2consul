PACKAGE_VERSION ?= $(shell git describe --always --tags)
REGISTRY ?= dockerhub.com/alleeclark/git2consul
ARGS ?= --consul-addr="172.17.0.1:8500" --git-url="https://github.com/alleeclark/test-git2consul.git" --git-branch="origin" sync
#USERNAME:=$(shell id -u -n)
#USERID:=$(shell id -u)
#--build-arg USERID=$(USERID) --build-arg USERNAME=$(USERNAME)

.PHONY: images
images:
	docker build -t git2consul:$(PACKAGE_VERSION) -f docker/Dockerfile ./

.PHONY: publish
publish: images
	docker tag git2consul:$(PACKAGE_VERSION) alleeclark/git2consul:latest 
	docker push alleeclark/git2consul:latest 

.PHONY: pull
pull:
	docker pull docker.io/alleeclark/git2consul:latest

.PHONY: devcluster
devcluster:
	docker run -itd --name consul --network=host consul:latest
	docker run -it git2consul:$(shell git describe --always --tags) $(ARGS)

.PHONY: refreshconsul
refreshconsul:
	docker restart consul

.PHONY: resync
resync:
	docker run -it --network=host git2consul:$(shell git describe --always --tags) --config-file="/var/git2consul/config.toml" resync