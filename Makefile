test:
	go clean -testcache
	GOFLAGS=-mod=vendor go test -v ./...

install:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.19.0
	curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh
	curl -sL https://git.io/autotag-install | sh
	GO111MODULE=off go get github.com/mattn/goveralls
	go mod tidy 
	go mod vendor

clean: 
	rm -rf ${PWD}/cover 

cover: clean 
	mkdir ${PWD}/cover 
	go clean -testcache
	GOFLAGS=-mod=vendor go test ./... -v -cover -coverprofile=${PWD}/cover/coverage.out

deploy-cover:
	goveralls -coverprofile=${PWD}/cover/coverage.out -service=circle-ci -repotoken=$$COVERALLS_TOKEN

bench: 
	GOFLAGS=-mod=vendor go test -bench=.  ./... -run=^B

lint: 
	./bin/golangci-lint run -c .golangci.yml ./...
	
lint-fix: 
	@FILES="$(shell find . -type f -name '*.go' -not -path "./vendor/*")"; goimports -local "github.com/shaj13/go-guardian/" -w $$FILES
	./bin/golangci-lint run -c .golangci.yml ./... --fix 
	./bin/golangci-lint run -c .golangci.yml ./... --fix

.SILENT: release
release: 
	git clean -df 
	git checkout -- .
	BRANCH=$(shell git rev-parse --abbrev-ref HEAD) \
	&& if [  $$BRANCH != "master" ]; then \
  		echo Aborting release on $$BRANCH; \
	else \
		bin/autotag --scheme=conventional; \
		bin/goreleaser release; \
	fi 