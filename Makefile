BINARY := enforcer
IMAGE := k8s-egress-policy-enforcer

.PHONY: build run test vet fmt tidy docker-build docker-run clean

build:
	go build -o bin/$(BINARY) ./cmd/enforcer

run: build
	./bin/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

tidy:
	go mod tidy

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker run --rm -p 8080:8080 $(IMAGE)

clean:
	rm -rf bin
