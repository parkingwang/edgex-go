BINARY?=$(shell cat name.txt)

IMAGE_TAG?=latest
IMAGE_ORG?=registry.cn-shenzhen.aliyuncs.com/nextabc
IMAGE_APP=${BINARY}

# Build opts
BUILD_ARCH=amd64
BUILD_ENV?=CGO_ENABLED=0 GOOS=linux GOARCH=${BUILD_ARCH}

all: build package.zip


build:
	@echo "Go BUILD ${BINARY}"
	@${BUILD_ENV} go build ${LD_FLAGS} -o ${BINARY}.raw .
	rm -f ${BINARY}
	upx -o ${BINARY} ${BINARY}.raw
	rm -f ${BINARY}.raw

package.zip: build
	zip -r package.zip ${BINARY} application.toml


image: build
	@echo "Docker IMAGE: $<"
	sudo docker build -t $(IMAGE_ORG)/$(IMAGE_APP):$(IMAGE_TAG) .

push: image
	@echo "Docker PUSH IMAGE: $<"
	sudo docker push $(IMAGE_ORG)/$(IMAGE_APP):$(IMAGE_TAG)

.PHONY: clean all
clean:
	rm -f ${BINARY} package.zip

