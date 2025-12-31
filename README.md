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
debug`. To run the server, you can either run `./build/git-server-poc` or launch it
from VSCode in debug mode.

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
- **Setup Command**: `microceph enable rgw`

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

**Check Cluster Status:**

```bash
./devenv/bin/microceph status
```

**Check Ceph Health:**

```bash
./devenv/bin/ceph -s
```

**List RGW Users:**

```bash
./devenv/bin/radosgw-admin user list
```

**Create RGW User:**

```bash
./devenv/bin/radosgw-admin user create --uid="hercules" --display-name="Hercules"
```

**User status:**

```bash
./devenv/bin/radosgw-admin user info --uid="hercules"
```

**Delete RGW User:**

```bash
./devenv/bin/radosgw-admin user rm --uid="hercules"
```
