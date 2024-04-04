.PHONY: install uninstall test-container-run

BINARY_NAME := netscout
SOURCES := $(wildcard *.go) $(wildcard osint/*.go) $(wildcard shared/*.go) $(wildcard dns/*.go)

install: $(SOURCES)
	go build -o $(BINARY_NAME) .
	mv $(BINARY_NAME) /usr/bin/

uninstall: $(SOURCES)
	rm -rf /usr/bin/$(BINARY_NAME)

# Pull DVWA Docker image
test-container-pull:
	docker pull citizenstig/dvwa

# Run the Docker container for DVWA
test-container-run:
	docker run -d -p 80:80 citizenstig/dvwa

testfiles-setup:
	zip -r ./testfiles/testdir.zip ./testfiles/testdir-src && echo 'test.com/valid\nexample.com/example\ntesting.com/wow' > ./testfiles/testxz.txt && xz ./testfiles/testxz.txt 

testfiles-teardown:
	rm -f ./testfiles/*.xz ./testfiles/*.zip

test:
	go test ./...
