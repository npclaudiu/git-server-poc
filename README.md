# Git Server PoC

This repository contains a proof of concept for a Git server using Ceph as
object storage and PostgreSQL as database. It currently explores only the data
ingestion process (the
[`git-receive-pack`](https://git-scm.com/docs/http-protocol#_smart_service_git_receive_pack)
protocol over HTTP).

## Quick Start

Prerequisites:

- [Go](https://golang.org/dl/)
- [Make](https://www.gnu.org/software/make/)
- [Docker](https://www.docker.com/)
- [Node.js](https://nodejs.org/)
- [pnpm](https://pnpm.io/)
- [dbmate](https://github.com/amacneil/dbmate)
- [sqlc](https://sqlc.dev/)

You can bootstrap the entire development environment using `make devenv`. This
command will set up a Docker environment with
[MicroCeph](https://github.com/canonical/microceph) for S3-compatible object
storage and [PostgreSQL](https://www.postgresql.org/) for metadata storage.

After the environment is up and running, you can build the server using `make debug`. To run the server, you can either run `./build/git-server` or
launch it from VSCode in debug mode.
