IMAGE_TAG?=latest
IMAGE_ORG?=yoojia
IMAGE_APP=edgex-endpoint-serial

# Ref to Dockerfile
BINARY=edgex-endpoint
# Build opts
BUILD_ARCH=amd64
BUILD_ENV?=CGO_ENABLED=0 GOOS=linux GOARCH=${BUILD_ARCH}

all: build package.zip


build: $(SRC)
	@echo "Go BUILD $@"
	@${BUILD_ENV} go build ${LD_FLAGS} -o ${BINARY}.raw.bin .
	rm -f ${BINARY}.bin
	upx -o ${BINARY}.bin ${BINARY}.raw.bin
	rm -f ${BINARY}.raw.bin


package.zip: build
	zip -r package.zip ${BINARY}.bin application.toml


image: build
	@echo "Docker IMAGE: $<"
	sudo docker build -t $(IMAGE_ORG)/$(IMAGE_APP):$(IMAGE_TAG) .

.PHONY: clean all
clean:
	rm *.bin *.zip

