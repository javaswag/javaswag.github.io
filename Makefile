.DEFAULT_GOAL := run

.PHONY: build
build:
	rm -rf docs/
	hugo --minify --buildDrafts --destination docs --baseURL=http://localhost:1313

.PHONY: preview
preview:
	python3 -m http.server 1313 --directory docs

.PHONY: run
run: build preview