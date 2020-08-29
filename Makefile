.PHONY: build-test

build-test:
	go build .
	./prioritile ./dataset/tiles_a dataset/tiles_b

