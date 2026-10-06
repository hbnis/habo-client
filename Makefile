.PHONY: run frontend fmt validate-fmt build-windows build-linux

run:
	scripts/run.sh

frontend:
	cd frontend && npm ci && npm run build

fmt:
	scripts/fmt.sh

validate-fmt:
	scripts/validate-fmt.sh

build-windows:
	scripts/build-windows.sh

build-linux:
	scripts/build-linux.sh
