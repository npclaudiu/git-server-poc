# Git Server PoC

> Disclaimer: This project is an experiment and there is no intent to make it
> production-ready.

## Introduction

This project is a proof of concept implementation of a custom Git server. It is
written in Go and relies on the [`go-git`](https://github.com/go-git/go-git)
library to implement the Git Smart HTTP protocol (`git-receive-pack`,
`git-upload-pack`). Tested to ensure that the implementation is compatible with
the de facto Git implementation.

Unlike traditional Git server implementations that rely on file-system-based
"bare" repositories, this server abstracts data persistence through custom
storage interfaces, routing data to specialized systems:

**Object Storage (Ceph RGW)**: Git objects (blobs, trees, commits) are
  content-addressed and stored in an S3-compatible object store. This approach
  addresses scalability challenges associated with massive file counts (the
  "Small File Problem") by treating Git objects as immutable data blobs.

**Relational Metadata (PostgreSQL)**: Mutable repository data, such as
  references (branches, tags), are managed in a relational database to ensure
  transactional consistency while queries remain efficient.

This project serves as an experimental sandbox for exploring ideas such as
the ones listed below, against standard Git workloads:

- [x] Implementing the Git/HTTP protocol with custom storage backends
- [ ] Using FastCDC for object data deduplication
- [ ] Using Merkle Trees for a multi-generational append-only object store
- [ ] Using swappable object and metadata stores

Developed with [Gemini Code Assist](https://codeassist.google/).

## Quick Start

The development environment is bootstrapped by a rather hacky set of scripts
contained in the `devenv` directory. For anything serious, I would recommend
using a more robust build system such as [Bazel](https://bazel.build/) instead.

### Prerequisites

- [Go](https://golang.org/dl/)
- [Make](https://www.gnu.org/software/make/)
- [Docker](https://www.docker.com/)
- [Node.js](https://nodejs.org/)
- [pnpm](https://pnpm.io/)
- [dbmate](https://github.com/amacneil/dbmate)
- [sqlc](https://sqlc.dev/)

### Setup

```bash
make devenv-up
```

This command will set up a Docker environment with
[MicroCeph](https://github.com/canonical/microceph) for S3-compatible object
storage and [PostgreSQL](https://www.postgresql.org/) for metadata storage.

### Build

```bash
make debug
```

### Run

```bash
./bin/git-server-poc
```

Alternatively, you can run it from an IDE in debug mode. VSCode configs are
already included.

## Application Structure

### REST API

The server provides a simple REST API for managing repositories.

- `POST /repositories`: Create a new repository.
  - Body: `{"name": "repo-name"}`
- `GET /repositories/{id}`: Get repository details.
- `PUT /repositories/{id}`: Update repository (e.g., rename).
- `DELETE /repositories/{id}`: Delete a repository.

### Git Smart HTTP

The server implements the standard Git Smart HTTP protocol, allowing standard
Git clients to interact with hosted repositories.

- `GET /repositories/{id}/info/refs`: Service discovery and reference
  advertisement.
- `POST /repositories/{id}/git-upload-pack`: Handles `git fetch` and `git clone`.
- `POST /repositories/{id}/git-receive-pack`: Handles `git push`.

### Implementation Details

This implementation deviates from standard directory-based Git servers in
several key ways:

#### Architecture

It uses a custom implementation of `go-git`'s `Storer` interface. This
abstracts the underlying storage, allowing us to route:

- **Objects** (blobs, trees, commits) to **Ceph** (via `internal/objectstore`).
- **References** (branches, tags) to **PostgreSQL** (via `internal/metastore`).

#### Object Storage

- Objects are stored as "loose objects" in S3-compatible Ceph buckets under the
    key pattern `repositories/{repo}/objects/{hash}`.
- The content is stored with the standard Git header (`type size\0`) prepended,
    allowing for compatibility and inspection.
- **Streaming Uploads**: To handle large pushes and avoid memory buffering
    issues, the server uses the AWS SDK's S3 Uploader. This enables streaming of
    packet-line data directly to Ceph without needing to seek the input stream.

#### Quirks & Workarounds

- **Manual Packet-Line Parsing**: During `git-receive-pack`, the server manually
  delimits the command packet-lines from the packfile data stream. This is
  necessary to prevent `go-git`'s default behavior from over-buffering or
  misinterpreting the stream boundaries when piping directly to object storage.

### Persistence

The server now implements persistence for repository state in S3-compatible
storage:

- **Objects**: Stored as `repositories/{repo}/objects/{hash}`.
- **Config**: Repository configuration is stored at
  `repositories/{repo}/config`.
- **Shallow Commits**: Shallow commit hashes are stored at
  `repositories/{repo}/shallow`.
- **Index**: The staging area (index) is stored at `repositories/{repo}/index`.

### Limitations

- **No Authentication**: The server is currently unprotected. Anyone can
  read/write to any repository.
- **Performance**: `IterEncodedObjects` (used for GC and some clones) lists keys
  via S3 API, which may be slow for large repositories.
- **No Packing**: Objects are stored strictly as loose objects. There is no
  support for generating or storing packfiles (.pack/.idx) for storage
  optimization.

## Ceph

MicroCeph is used to provide an S3-compatible object storage service (RGW) for
storing Git objects. The entire Ceph cluster runs inside a single Docker
container named `microceph`.

### Cluster Topology

The Ceph cluster is configured as a single-node cluster with the following
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

Since Ceph runs inside a Docker container using `snapd`, the binaries are
installed in `/snap/bin/`. You can invoke them from the host machine using
`docker exec`. The generic pattern to run any Ceph command is:

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

# Bucket stats:
./devenv/bin/radosgw-admin bucket stats --bucket=git-objects
```

## PostgreSQL

PostgreSQL is used for metadata storage (everything but objects and packs). It
runs in a Docker container named `postgres`.

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

__

Copyright © 2026, Claudiu Nedelcu. All rights reserved. Licensed under the [MIT
License](LICENSE.txt).
