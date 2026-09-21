TOP_SRCDIR = ../..
PACKAGE = habo-client
PACKAGE_NAME = Habo Client
PACKAGE_BUGREPORT = https://github.com/hbnis/habo-client/issues
INSTALLER_COMPRESSION = lzma
PACKAGE_EXE = habo-client.exe

all: nsis

nsis:
	set -e; if echo "${GITHUB_REF_NAME}" | grep -qE '^v?[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		export VERSION=$$(echo "${GITHUB_REF_NAME}" | sed 's/^v//'); \
	else \
		export VERSION="0.0.1"; \
	fi; \
	makensis -DPACKAGE_VERSION="$$VERSION" -DPACKAGE="$(PACKAGE)" -DPACKAGE_NAME="$(PACKAGE_NAME)" -DPACKAGE_EXE="$(PACKAGE_EXE)" -DPACKAGE_BUGREPORT="$(PACKAGE_BUGREPORT)" -DTOP_SRCDIR="$(TOP_SRCDIR)" -DOUTFILE=../../habo-client-amd64-installer.exe -X"SetCompressor /FINAL /SOLID $(INSTALLER_COMPRESSION)" albiondata-client.nsi $<
