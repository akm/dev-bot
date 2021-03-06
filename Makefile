# GAE_PROJECT ?= example-project
BASEDIR = $(CURDIR)
VERSION ?= $(shell cat ./VERSION)
LOCAL_PORT ?= 8080

SERVICE_NAME=dev-bot
DEPLOY_PATH=.

.PHONY: setup
setup:
	@which dep || go get -u github.com/golang/dep/cmd/dep

.PHONY: init
init: setup Gopkg.toml .gitignore

Gopkg.toml:
	@dep init
.gitignore:
	@echo "/vendor" > .gitignore

.PHONY: dep_ensure
dep_ensure:
	@dep ensure

.PHONY: build
build:
	goapp build .

app.yaml:
	erb -T - app.yaml.erb > ./app.yaml || rm app.yaml

.PHONY: deploy rollback
deploy: app.yaml
	appcfg.py -A $(GAE_PROJECT) -V ${VERSION} update $(DEPLOY_PATH); rm app.yaml
rollback:
	appcfg.py -A $(GAE_PROJECT) -V ${VERSION} rollback $(DEPLOY_PATH)

.PHONY: update-traffic
update-traffic:
	gcloud --project ${GAE_PROJECT} app services set-traffic ${SERVICE_NAME} --splits=${VERSION}=1 -q

.PHONY: local_http_server
local_server:
	dev_appserver.py \
		--port=$(LOCAL_PORT) \
		$(BASEDIR)/app.yaml

.PHONY: local
local: local_http_server

.PHONY: dev
dev: build local
