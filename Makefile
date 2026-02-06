GIT_VERSION = $(shell git describe --tags --always)
FLAGS = -ldflags "\
  -X config/config.VERSION=$(GIT_VERSION) \
"

run:
	STATICS=statics/www/ go run $(FLAGS) .

build:
	go build $(FLAGS) -o bin/ .

test:
	go test -count=1 -cover ./...

dep:
	go mod tidy
	go mod vendor

cloc:
	cloc --exclude-dir=vendor,data .

version:
	@echo "${GIT_VERSION}"
