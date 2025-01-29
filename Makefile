.DEFAULT_GOAL := run

.PHONY: build
build:
	rm -rf docs/
	hugo  --buildDrafts --destination docs --baseURL=http://localhost:1313 && npm run build

.PHONY: preview
preview:
	python3 -m http.server 1313 --directory docs

.PHONY: run
run: build preview