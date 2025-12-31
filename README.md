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

After the environment is up and running, you can build the server using `make
debug`. To run the server, you can either run `./build/git-server-poc` or launch
it from VSCode in debug mode.

## Ceph

MicroCeph is used to provide an S3-compatible object storage service (RGW) for
storing Git objects. The entire Ceph cluster runs inside a single privileged
Docker container named `microceph`.

### Cluster Topology

The MicroCeph cluster is configured as a single-node cluster with the following
components:

- **Monitors (MONs)**: 1 (Integrated single-node default)
- **Managers (MGRs)**: 1 (Integrated single-node default)
- **Metadata Servers (MDSs)**: 1 (Enabled during bootstrap)
- **OSDs (Object Storage Daemons)**: 3
  - Type: Loopback file-backed
  - Size: 4GB each
  - Configuration Command: `microceph disk add loop,4G,3`

### RGW Setup

The Rados Gateway (RGW) is enabled to provide the S3 API.

- **Service Name**: `rgw`
- **Port**: 8000 (Mapped to host port 8000)

A default user is automatically created by the `devenv` script to generate
S3-like RGW access and secret keys, which are then written to `config.yaml`.

### Invoking Binaries

Since MicroCeph runs inside a Docker container using `snapd`, the binaries are
installed in `/snap/bin/`. You can invoke them from the host machine using
`docker exec`. The generic pattern to run any MicroCeph command is:

```bash
docker exec microceph /snap/bin/<binary_name> [arguments]
```

However, for the most common commands, this project provides wrapper scripts in
`./devenv/bin`.

### Common Commands

```bash
# Check Cluster Status:
./devenv/bin/microceph status

# Check Ceph Health:
./devenv/bin/ceph -s

# List RGW Users:
./devenv/bin/radosgw-admin user list

# Create RGW User:
./devenv/bin/radosgw-admin user create --uid="hercules" --display-name="Hercules"

# User status:
./devenv/bin/radosgw-admin user info --uid="hercules"

# Delete RGW User:
./devenv/bin/radosgw-admin user rm --uid="hercules"
```

## PostgreSQL

PostgreSQL is used for metadata storage (repositories, etc.). It runs in a
Docker container named `postgres`.

### Configuration

- **Version**: 18.1
- **Port**: 5432 (Mapped to host port 5432)
- **Database**: `git-server-poc`
- **User**: `minerva`
- **Password**: `m1n3rv@`

### Schema Management

We use [dbmate](https://github.com/amacneil/dbmate) for database schema
migrations.

**Run Migrations:**

```bash
make ms-migrate
```

### Code Generation

We use [sqlc](https://sqlc.dev/) to generate Go code from SQL queries.

**Generate Code:**

```bash
make ms-gen
```

### Common Commands

```bash
# Connect to database in interactive mode:
./devenv/bin/psql

# List databases:
./devenv/bin/psql -l

# List tables:
./devenv/bin/psql -c "\dt"

# List columns of a table:
./devenv/bin/psql -c "\d+ table_name"

# List indexes of a table:
./devenv/bin/psql -c "\di+ table_name"

# List sequences of a table:
./devenv/bin/psql -c "\ds+ table_name"
```
