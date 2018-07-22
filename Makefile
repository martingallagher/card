.PHONY: build run

get_dep:
	command -v dep || go get -u github.com/golang/dep/cmd/dep

update_deps: get_dep
	go get -u ./...
	rm -rf Gopkg.* vendor/
	dep init

ensure_deps:
	dep ensure --vendor-only

test:
	go test -v -cover -failfast ./...

build:
	go build ./service/api

run: build
	./api
