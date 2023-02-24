GOFLAGS=-v
RUNFLAGS=-i

build: tidy
	go build ${GOFLAGS} -o eclass-tui

tidy:
	go mod tidy
	@touch tidy

run:
	go run main.go ${RUNFLAGS}

.PHONY: run build # not tidy
