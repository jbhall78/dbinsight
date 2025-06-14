PROJECT_NAME = dbinsight
VERSION = 0.1.3

all: $(PROJECT_NAME)-proxy $(PROJECT_NAME)-create-db

$(PROJECT_NAME)-proxy:
	go mod tidy
	go build -o $(PROJECT_NAME)-proxy ./cmd/proxy

$(PROJECT_NAME)-create-db:
	go mod tidy
	go build -o $(PROJECT_NAME)-create-db ./cmd/create-db

output-qemu/$(PROJECT_NAME)-proxy-qemu:
	packer build packer/qemu/template.json

qemu-build: output-qemu/$(PROJECT_NAME)-proxy-qemu

docker-build:
	DOCKER_BUILDKIT=1 docker build -t $(PROJECT_NAME)-proxy:$(VERSION) -f docker/images/proxy/Dockerfile .

docker-run:
	docker run -d --name dbinsight-proxy --restart always --net=host -p 3306:3306 $(PROJECT_NAME)-proxy:$(VERSION)

#test:
#	go test ./internal/... ./pkg/... # Run tests

clean:
	rm -f $(PROJECT_NAME)-proxy
	rm -f $(PROJECT_NAME)-create-db

qemu-clean:
	rm -rf output-qemu

.PHONY: clean docker-build docker-run
