test_fmt:
	@echo Checking correct formatting of files...
	@{ \
		files=$$( go fmt ./... ); \
		if [ -n "$$files" ]; then \
		echo "Files not properly formatted: $$files"; \
		exit 1; \
		fi; \
	}

build:
	@echo Testing build...
	@{ \
		go build sda/app/prifi.go && rm -f prifi; \
	}

test_govet:
	@echo Running go vet...
	@{ \
		if ! go vet ./...; then \
		exit 1; \
		fi \
	}

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

coveralls:
	./coveralls.sh

test_verbose:
	go test -v -race -short ./...

it:
	./prifi.sh integration-test || cat relay.log

it2:
	./prifi.sh integration-test2

clean:
	rm -f profile.cov *.log

test: build test_fmt test_govet test_lint

all: test coveralls it
