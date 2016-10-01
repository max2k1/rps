GOTOOLS = \
	github.com/max2k1/render_number

# all builds binaries for all targets
all: tools
	@go build -o rps

distclean: clean

clean:
	@rm rps 2>/dev/null || true

tools:
	@go version
	@go get -u -v $(GOTOOLS)
