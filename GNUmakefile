GOTOOLS = \
	github.com/max2k1/render_number

# all builds binaries for all targets
all: tools
	@go build

clean:
	@rm rps || true

tools:
	go get -u -v $(GOTOOLS)
