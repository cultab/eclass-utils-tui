GOFLAGS=-v
RUNFLAGS=-i

build: tidy
	go build ${GOFLAGS} -o eclass-tui

tidy:
	go mod tidy
	@touch tidy

clean:
	rm -f ./eclass-tui ./tidy

run: build
	./eclass-tui ${RUNFLAGS}

.PHONY: run build clean # not tidy
