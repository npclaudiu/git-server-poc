EXECUTABLE=git-server-poc

.PHONY: all
all: build

.PHONY: debug
debug:
	go build -v -o bin/$(EXECUTABLE) ./cmd/$(EXECUTABLE)/main.go

.PHONY: release
release:
	go build -v -o bin/$(EXECUTABLE) -ldflags="-s -w" -trimpath \
		./cmd/$(EXECUTABLE)/main.go

.PHONY: build
build: debug

.PHONY: test
test:
	go test ./...

.PHONY: devenv_vm_setup
devenv_vm_setup:
	./scripts/devenv_vm_setup.sh

.PHONY: devenv_vm_clean
devenv_vm_clean:
	multipass delete devenv
	multipass purge

.PHONY: ceph_setup
ceph_setup:
	./scripts/ceph_setup.sh

.PHONY: ceph_user
ceph_user:
	./scripts/ceph_user.sh

.PHONY: ceph_status
ceph_status:
	multipass exec devenv -- sudo microceph status

.PHONY: pg_setup
pg_setup:
	./scripts/pg_setup.sh

.PHONY: pg_status
pg_status:
	multipass exec devenv -- systemctl status postgresql

.PHONY: devenv
devenv: devenv_vm_setup
	make ceph_setup
	make ceph_user
	make pg_setup

.PHONY: format
format:
	go fmt ./...

.PHONY: clean
clean: devenv_vm_clean
	rm -rf bin
