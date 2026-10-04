.PHONY: run frontend fmt validate-fmt build-windows build-linux build-darwin

run:
	scripts/run.sh

frontend:
	cd frontend && npm ci && npm run build

fmt:
	scripts/fmt.sh

validate-fmt:
	scripts/validate-fmt.sh

build-windows:
	go mod tidy
	scripts/build-windows.sh

build-linux:
	go mod tidy
	scripts/build-linux.sh

build-darwin:
	go mod tidy
	scripts/build-darwin.sh
