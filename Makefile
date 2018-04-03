.PHONY: all
all: test coveralls it it2

.PHONY: test
test: build test_fmt test_govet test_lint
	DEBUG_COLOR="True" DEBUG_LVL=1 go test -short ./...

.PHONY: test_fmt
test_fmt:
	@echo Checking correct formatting of files...
	@{ \
		files=$$( go fmt ./... ); \
		if [ -n "$$files" ]; then \
		echo "Files not properly formatted: $$files"; \
		exit 1; \
		fi; \
	}

.PHONY: test_fmt
build:
	@echo Testing build...
	@{ \
		go build sda/app/prifi.go && rm -f prifi; \
	}

.PHONY: test_govet
test_govet:
	@echo Running go vet...
	@{ \
		if ! go vet ./...; then \
		exit 1; \
		fi \
	}

.PHONY: coveralls
coveralls:
	./coveralls.sh

.PHONY: test_verbose
test_verbose:
	DEBUG_COLOR="True" DEBUG_LVL=3 go test -v -race ./...

.PHONY: it
it:
	./test.sh integration || (cat relay.log; exit 1)

.PHONY: it2
it2:
	./test.sh integration2 || (cat relay.log; exit 1)

.PHONY: clean
clean:
	rm -f profile.cov *.log timing.txt prifi-lib/relay/timing.txt

.PHONY: test_lint
test_lint:
	@echo Checking linting of files ...
	@{ \
		go get -u github.com/golang/lint/golint; \
		exclude="_test.go|ALL_CAPS|underscore|should be of the form|.deprecated|and that stutters|error strings should not be capitalized"; \
		lintfiles=$$( golint ./... | egrep -v "($$exclude)" ); \
		if [ -n "$$lintfiles" ]; then \
		echo "Lint errors:"; \
		echo "$$lintfiles"; \
		exit 1; \
		fi \
	}
