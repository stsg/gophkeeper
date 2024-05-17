B=$(shell git rev-parse --abbrev-ref HEAD)
BRANCH=$(subst /,-,$(B))
GITREV=$(shell git describe --abbrev=7 --always --tags)
REV=$(GITREV)-$(BRANCH)-$(shell date +%Y%m%d-%H:%M:%S)

info:
	- @echo "revision $(REV)"

build: info
	@ echo
	@ echo "Compiling Binary"
	@ echo
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.revision=$(REV) -s -w" -o bin/gophkeeper app/main.go

docker:
	docker build -t starky/gophkeeper:master .

clean:
	@ echo
	@ echo "Cleaning"
	@ echo
	rm bin/gophkeeper

tidy:
	@ echo
	@ echo "Tidying"
	@ echo
	go mod tidy

run:
	go run -ldflags "-X main.revision=$(REV) -s -w" app/main.go --dbg

lint:
	@ echo
	@ echo "Linting"
	@ echo
	golangci-lint run

test:
	@ echo
	@ echo "Testing"
	@ echo
	go test

.PHONY: *

