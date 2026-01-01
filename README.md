# Git Server PoC

> Disclaimer: This project is an experiment in an early stage of development and
> there is no intent to make it production-ready. The documentation is
> incomplete and the code is subject to change.

## Introduction

This project is a Proof of Concept implementation of a custom Git server written
in Go, designed to explore cloud-native storage architectures for high-scale
version control systems. It leverages the
[`go-git`](https://github.com/go-git/go-git) library to implement the Git Smart
HTTP protocol (`git-receive-pack`, `git-upload-pack`), enabling seamless
interaction with standard Git clients. Extensive testing is continuously
performed to ensure that the implementation is compatible with the de facto Git
implementation.

Unlike traditional Git server implementations that rely on file-system-based
"bare" repositories, this server abstracts data persistence through custom
storage interfaces, routing data to specialized systems:

- **Object Storage (S3/Ceph)**: Git objects (blobs, trees, commits) are
  content-addressed and stored in an S3-compatible object store. This approach
  addresses scalability challenges associated with massive file counts (the
  "Small File Problem") by treating Git objects as immutable data blobs.
- **Relational Metadata (PostgreSQL)**: Mutable repository data, such as
  references (branches, tags) and access control lists, are managed in a
  relational database to ensure transactional consistency and efficient
  queryability.

This architecture allows for independent scaling of storage and compute
resources, as well as advanced data analysis opportunities such as global
deduplication (e.g., via FastCDC) and Merkle Tree validation. The project serves
as an experimental sandbox to validate these patterns against standard Git
workloads.

Developed with [Gemini Code Assist](https://codeassist.google/).

## Quick Start

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
make devenv
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
- `POST /repositories/{id}/git-upload-pack`: Handles `git fetch` and `git clone`
  (Logic currently stubbed).
- `POST /repositories/{id}/git-receive-pack`: Handles `git push`.

### Implementation Details

This implementation deviates from standard directory-based Git servers in
several key ways:

- **Architecture**: It uses a custom implementation of `go-git`'s
  `storer.Storer` interface. This abstracts the underlying storage, allowing us
  to route:
  - **Objects** (blobs, trees, commits) to **S3** (via `internal/objectstore`).
  - **References** (branches, tags) to **PostgreSQL** (via
    `internal/metastore`).

- **Object Storage**:
  - Objects are stored as "loose objects" in S3 buckets under the key pattern
    `repositories/{repo}/objects/{hash}`.
  - The content is stored with the standard Git header (`type size\0`)
    prepended, allowing for compatibility and inspection.
  - **Streaming Uploads**: To handle large pushes and avoid memory buffering
    issues, the server uses the AWS SDK's `feature/s3/manager` Uploader. This
    enables streaming of packet-line data directly to S3 without needing to seek
    the input stream.

- **Quirks & Workarounds**:
  - **Manual Packet-Line Parsing**: During `git-receive-pack`, the server
    manually delimits the command packet-lines from the packfile data stream.
    This is necessary to prevent `go-git`'s default behavior from over-buffering
    or misinterpreting the stream boundaries when piping directly to object
    storage.

### Limitations

- **No Authentication**: The server is currently unprotected. Anyone can
  read/write to any repository.
- **Incomplete Storer Implementation**:
  - `IterEncodedObjects` is not yet implemented. This limits operations that
    require full object traversal, such as garbage collection or complete
    packfile generation for clones.
  - Configuration, Index, and Shallow storage methods are currently stubs.
- **No Packing**: Objects are stored strictly as loose objects. There is no
  support for generating or storing packfiles (.pack/.idx) for storage
  optimization.

## Ceph

MicroCeph is used to provide an S3-compatible object storage service (RGW) for
storing Git objects. The entire Ceph cluster runs inside a single Docker
container named `microceph`.

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
