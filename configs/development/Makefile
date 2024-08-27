.PHONY: build docker-build down up ensure-consul
all: build ensure-consul docker-build down up

CONSUL_VERSION := $(shell curl -sL https://releases.hashicorp.com/consul/ | grep -Eo 'consul_[0-9]+\.[0-9]+\.[0-9]+' | sort -V | tail -1 | cut -d'_' -f2)
CONSUL_ZIP := consul_$(CONSUL_VERSION)_linux_arm64.zip

build:
	GOOS=linux GOARCH=arm64 go build -o tagit -ldflags="-s -w" ../../

ensure-consul:
	@if [ ! -f "$(CONSUL_ZIP)" ]; then \
		echo "Downloading Consul version $(CONSUL_VERSION)"; \
		curl -sLO "https://releases.hashicorp.com/consul/$(CONSUL_VERSION)/$(CONSUL_ZIP)"; \
	else \
		echo "Consul zip file already exists (version $(CONSUL_VERSION))"; \
	fi
	@echo "Extracting Consul binary"
	@unzip -o "$(CONSUL_ZIP)" consul
	@chmod +x consul

docker-build:
	docker build -f Dockerfile.tagit -t ncode/tagit:dev .
	docker build -f Dockerfile.consul -t ncode/consul:dev .

up:
	docker compose up

down:
	docker compose down

clean:
	rm -f tagit consul $(CONSUL_ZIP)