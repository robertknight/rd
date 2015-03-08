all: *.go
	go build

BIN_DIR=${DESTDIR}/usr/bin
SHARE_DIR=${DESTDIR}/usr/share/rd

install:
	install -d ${BIN_DIR}
	install rd ${BIN_DIR}
	install -d ${SHARE_DIR}
	install integration/* ${SHARE_DIR}
