PROJECT_NAME = dbinsight
VERSION = 0.1.0

all: $(PROJECT_NAME)-proxy $(PROJECT_NAME)-create-db output-qemu/$(PROJECT_NAME)-proxy-qemu

$(PROJECT_NAME)-proxy:
	go build -o $(PROJECT_NAME)-proxy ./cmd/proxy

$(PROJECT_NAME)-create-db:
	go build -o $(PROJECT_NAME)-create-db ./cmd/create-db

#test:
#	go test ./internal/... ./pkg/... # Run tests

docker-build:
	docker build -t $(PROJECT_NAME)-proxy:$(VERSION) .

docker-run:
	docker run -d --restart always -p 3306:3306 $(PROJECT_NAME)-proxy:$(VERSION)

clean:
	rm -f $(PROJECT_NAME)-proxy
 
output-qemu/$(PROJECT_NAME)-proxy-qemu:
	packer build packer/qemu/template.json

.PHONY: clean docker-build docker-run
