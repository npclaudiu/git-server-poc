EXECUTABLE=git-server-poc

.PHONY: all
all: test build

.PHONY: build
build:
	go build -v -o bin/$(EXECUTABLE) -ldflags="-s -w \
		./cmd/$(EXECUTABLE)/main.go

.PHONY: test
test:
	go test ./...

.PHONY: devenv_vm_setup
devenv_vm_setup:
	./scripts/devenv_vm_setup.sh

.PHONY: devenv_vm_clean
devenv_vm_clean:
	multipass delete ceph-dev
	multipass purge

.PHONY: ceph_setup
ceph_setup:
	./scripts/ceph_setup.sh

.PHONY: ceph_user
ceph_user:
	./scripts/ceph_user.sh

.PHONY: ceph_status
ceph_status:
	multipass exec ceph-dev -- sudo microceph status

.PHONY: pg_setup
pg_setup:
	./scripts/pg_setup.sh

.PHONY: pg_status
pg_status:
	multipass exec ceph-dev -- systemctl status postgresql

.PHONY: devenv
devenv: devenv_vm_setup
	make ceph_setup
	make ceph_user
	make pg_setup

.PHONY: clean
clean: devenv_vm_clean
	rm -f bin
