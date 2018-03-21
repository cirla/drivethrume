.PHONY : all build clean frontend backend run _run run_site run_funcs
.DEFAULT_GOAL := all

all : build

build : frontend backend

clean:
	rm lambda/api.zip
	rm -f functions/*

frontend:
	yarn build

backend:
	mkdir -p functions
	cd lambda && \
		go get ./... && \
		GOOS=linux go build -o ../functions/find_drivethrus ./find_drivethrus.go
	zip -j lambda/api.zip functions/*

run:
	$(MAKE) -j2 _run
_run: run_site run_funcs

run_site:
	yarn dev-server

run_funcs:
	yarn dev-api
