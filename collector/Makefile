BASE_SPACE:=$(shell pwd)
BUILD_SPACE:=$(BASE_SPACE)/build
BUCKET_NAME:=lambda-artifacts-$(shell dd if=/dev/random bs=8 count=1 2>/dev/null | od -An -tx1 | tr -d ' \t\n')
LAYER_NAME:=otel-collector

VERSION=$(shell cat VERSION)
GIT_SHA=$(shell git rev-parse HEAD)
GOARCH ?= amd64
GOBUILD=GO111MODULE=on CGO_ENABLED=0 installsuffix=cgo go build -trimpath
BUILD_INFO_IMPORT_PATH=main

LDFLAGS=-ldflags "-s -w -X $(BUILD_INFO_IMPORT_PATH).GitHash=$(GIT_SHA) -X $(BUILD_INFO_IMPORT_PATH).Version=$(VERSION) \
-X github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsxrayexporter.collectorDistribution=opentelemetry-collector-lambda"

.SILENT:
clean:
	rm -rf build/

build: clean
	@echo 👉 Building otel collector extension
	mkdir -p $(BUILD_SPACE)/extensions
	GOOS=linux $(GOBUILD) $(LDFLAGS) -o $(BUILD_SPACE)/extensions .

package: build
	@echo 👉 Package zip file for collector extension layer
	if [ ! $(certs-path) ]; then echo "Please input certs-path";exit 1; fi
	if [ ! -d $(certs-path) ]; then echo "This folder is not exist: $(certs-path) ";exit 1; fi
	if [ -a $(certs-path)/ca.pem ]; then echo "Check file ca.pem";else echo "ca.pem is not exist";exit 1; fi;
	if [ -a $(certs-path)/agent.pem ]; then echo "Check file agent.pem";else echo "agent.pem is not exist";exit 1; fi;
	if [ -a $(certs-path)/agent-key.pem ]; then echo "Check file agent-key.pem";else echo "agent-key.pem is not exist";exit 1; fi;
	if [ -a $(certs-path)/config.yaml ]; then echo "Check file config.yaml";else echo "config.yaml is not exist";exit 1; fi;
	mkdir -p $(BUILD_SPACE)/collector-config
	cp -R $(certs-path)/* $(BUILD_SPACE)/collector-config
	cd $(BUILD_SPACE) && zip -r collector-extension.zip collector-config extensions

publish-layer: package
	@echo 👉 Publishing collector extension layer...
	aws s3 mb s3://$(BUCKET_NAME)
	aws s3 cp $(BUILD_SPACE)/collector-extension.zip s3://$(BUCKET_NAME)

ifdef compatible-architecture
	aws lambda publish-layer-version --layer-name $(LAYER_NAME) --content S3Bucket=$(BUCKET_NAME),S3Key=collector-extension.zip --compatible-runtimes go1.x provided.al2 --compatible-architecture ${compatible-architecture} --query 'LayerVersionArn' --output text
else
	aws lambda publish-layer-version --layer-name $(LAYER_NAME) --content S3Bucket=$(BUCKET_NAME),S3Key=collector-extension.zip --compatible-runtimes go1.x provided.al2 --query 'LayerVersionArn' --output text
endif

	@echo 👉 Clearing cached files...
	aws s3 rm s3://$(BUCKET_NAME)/collector-extension.zip
	aws s3 rb s3://$(BUCKET_NAME)
	@echo 🚀 OpenTelemetry Collector layer published.