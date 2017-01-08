test_fmt:
	@echo Checking correct formatting of files...
	@{ \
		files=$$( go fmt ./... ); \
		if [ -n "$$files" ]; then \
		echo "Files not properly formatted: $$files"; \
		exit 1; \
		fi; \
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
		exclude="_test.go|ALL_CAPS|underscore|should be of the form|.deprecated|and that stutters|error strings should not be capitalized|composite literal uses unkeyed fields"; \
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

integration_test:
	./prifi.sh integration-test


test: test_fmt test_govet test_lint

all: test coveralls integration_test
