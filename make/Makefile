BINARY?=$(shell cat name.txt)

# Build opts
BUILD_ARCH=$(shell echo ${OSARCH})
BUILD_ENV?=CGO_ENABLED=0 GOOS=linux GOARCH=${BUILD_ARCH}

IMAGE_TAG?=201905.1-${BUILD_ARCH}
IMAGE_ORG?=registry.cn-shenzhen.aliyuncs.com/nextabc
IMAGE_NAME=${IMAGE_ORG}/${BINARY}:${IMAGE_TAG}


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
	sudo docker build -t ${IMAGE_NAME} .

push:
	@echo "Docker PUSH IMAGE: $<"
	sudo docker push ${IMAGE_NAME}

.PHONY: clean all
clean:
	rm -f ${BINARY} package.zip

