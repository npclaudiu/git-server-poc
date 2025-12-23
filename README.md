# Git Server PoC

This repository contains a proof of concept for a Git server using Ceph as
object storage and PostgreSQL as database. It currently explores only the data
ingestion process (the [`git-receive-pack` protocol over
HTTP](https://git-scm.com/docs/http-protocol#_smart_service_git_receive_pack)).

## Quick Start

Prerequisites: [Go](https://golang.org/dl/),
[Make](https://www.gnu.org/software/make/), [Multipass](https://multipass.run/).

You can bootstrap the entire development environment using `make devenv`. This
command will set up a VM with
[MicroCeph](https://github.com/canonical/microceph) for S3-compatible object
storage and [PostgreSQL](https://www.postgresql.org/) for metadata storage.
