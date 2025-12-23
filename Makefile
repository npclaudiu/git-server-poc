.PHONY: all
all: devenv

.PHONY: ceph_vm
ceph_vm:
	./scripts/ceph_vm.sh

.PHONY: ceph_setup
ceph_setup:
	./scripts/ceph_setup.sh

.PHONY: ceph_user
ceph_user:
	./scripts/ceph_user.sh

.PHONY: ceph_status
ceph_status:
	multipass exec ceph-dev -- sudo microceph status

.PHONY: ceph_clean
ceph_clean:
	multipass delete ceph-dev
	multipass purge

.PHONY: pg_setup
pg_setup:
	./scripts/pg_setup.sh

.PHONY: pg_status
pg_status:
	multipass exec ceph-dev -- systemctl status postgresql

.PHONY: devenv
devenv: ceph_vm
	make ceph_setup
	make ceph_user
	make pg_setup

.PHONY: clean
clean: ceph_clean
