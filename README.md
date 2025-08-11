# alf-cli

**Alfresco Docker CLI** — a small, opinionated command‑line tool that scaffolds a ready‑to‑run Docker Compose stack for **Alfresco Content Services (ACS)** in minutes.

> Status: early but usable. Expect breaking changes while the CLI stabilizes.

## Why

If you spin up ACS regularly—for demos, labs, PoCs, or local dev, you know the drill: wire up repository, DB, Search, transforms, proxy, pick ports, tweak memory, maybe add SMTP/LDAP/ActiveMQ, and cross‑locale indexing... This CLI turns that ritual into a guided wizard (or a single non‑interactive command) that outputs a clean, customizable Compose workspace.

> This project is intended to be a replacement for [https://github.com/alfresco/alfresco-docker-installer](https://github.com/alfresco/alfresco-docker-installer) and can be executed without additional requirements, as it is released as a native binary for Linux, macOS, and Windows.


## Highlights

* **Interactive wizard** with sensible defaults (single‑question flow; remaps your choices as you go).
* **Non‑interactive mode** for CI/scripts with flags for every prompt.
* **Resource‑aware templates**: scales service CPU/RAM limits from available Docker resources.
* **Multiple ACS versions** (e.g., 25.2, 25.1) with per‑version adjustments.
* **Optional components**: MariaDB or Postgres, ActiveMQ, SMTP, LDAP, FTP.
* **Search Services** (Solr) with **HTTP/HTTPS** comms and cross‑locale/content indexing toggles.
* **HTTPS toggle** for the public proxy; custom server name and port.
* **Add‑ons**: include selected community JARs/AMPs into the repo image.
* **Volumes**: choose Docker named volumes or bind mounts; optional volume bootstrap script.
* **Generated README** inside the output folder with credentials, endpoints, and maintenance tips.

## Prerequisites

* **Docker** (Desktop or Engine). Recent Desktop (>= 4.x) recommended.
* **Docker Compose v2** (`docker compose ...`).
* **Go 1.21+** (for building from source), or a prebuilt binary when releases are published.

> **Note (Docker Desktop 4.44+)**: the default builder is containerized Buildx. If you build service images locally and need them in your host image store, add `--load` to `docker build`. Example: `docker build --load -t my-image .`

## Install

Download the binary compiled for your architecture (Linux, Windows or Mac OS) from [Releases](https://github.com/aborroy/alf-cli/releases).

You may rename the binary to `alf`, all the following samples are using this command name by default.

Using `-h` flag provides detail on the use of the different commands available.

## Quick start (interactive)

This creates a new folder with a complete Compose workspace.

```bash
alf docker-compose
```

You’ll be prompted for:

* **ACS version** (e.g., 25.2 or 25.1)
* **HTTPS** (public proxy)
* **Server name** (default `localhost`)
* **Admin password** (`admin` user)
* **HTTP port** (single port exposed by proxy)
* **Bind to IP** (optional)
* **Database** (Postgres / MariaDB)
* **Search** options (HTTP/HTTPS, cross‑locale, content indexing)
* **ActiveMQ**, **SMTP**, **LDAP**, **FTP** toggles
* **Add‑ons** selection
* **Volumes** strategy

When it finishes, you’ll see a friendly summary of resources detected and the files written.

Start the stack:

```bash
cd <your-output-folder>
docker compose up -d
```

Watch logs (helpful on first boot):

```bash
docker compose logs -f alfresco
```

## Non‑interactive usage

Every prompt has a corresponding flag so you can script it. See built‑in help:

```bash
alf docker-compose --help
```

**Example** (illustrative; adjust flags to match `--help`):

```bash
alf docker-compose \
  --version 25.1 \
  --https=false \
  --server localhost \
  --admin-password 's3cret' \
  --http-port 8080 \
  --db mariadb \
  --solr-comms https \
  --index-cross-locale=true \
  --index-content=true \
  --activemq=false \
  --smtp=false \
  --ldap=false \
  --ftp=false \
  --addons alf-tengine-ocr \
  --use-docker-volume=true
```

## What gets generated

A tidy workspace you can version‑control as needed. Typical tree:

```
.
├── .env
├── compose.yaml                
├── README.md                   
├── config/
│   └── nginx.conf
├── alfresco/
│   ├── Dockerfile
│   └── modules/
│       ├── jars/
│       └── amps/
├── share/
│   └── Dockerfile
├── search/
│   └── Dockerfile
├── scripts/
│   └── create-volumes.sh      
└── keystores/                  
```

> Exact files depend on your selections (DB, addons, HTTPS, volumes, etc.).

## Endpoints & credentials

* **Repository (REST):** `http://<server>:<port>/alfresco`
* **Share:** `http://<server>:<port>/share`
* **Content App:** `http://<server>:<port>/content-app`
* **Control Center App:** `http://<server>:<port>/admin`
* **Admin user:** `admin`
* **Admin password:** the value you chose during generation

## Day‑2 operations

**Stop / Start**

```bash
docker compose down
docker compose up -d
```

**Logs**

```bash
docker compose logs -f
```

**Reconfigure / Upgrade**

* Re‑run the CLI with a new **ACS version** or toggles.
* Keep volumes if you want content and DB data to persist; prune them to start clean.

**Backup (quick‑n‑dirty dev)**

> Convenience backups for a local dev stack. For production, prefer proper DB dumps/snapshots and tested restore procedures.

### Stop the stack (for volume copies)

```bash
docker compose down
```

### If you used Docker **named volumes**

Back up a named volume to `./backups/` using a throwaway container (OS‑agnostic):

```bash
mkdir -p backups && docker run --rm -v <volume-name>:/data:ro -v "$PWD/backups":/backup alpine sh -c 'cd /data && tar -czf /backup/<volume-name>-$(date +%F).tgz .'
```

Restore the archive into a (new or empty) volume:

```bash
docker volume create <volume-name> && docker run --rm -v <volume-name>:/data -v "$PWD/backups":/backup alpine sh -c 'cd /data && tar -xzf /backup/<volume-archive>.tgz'
```

### If you used **bind‑mounted directories**

Compress/copy the host directories referenced under `volumes:` in `compose.yaml`.

```bash
mkdir -p backups && tar -czf backups/<name>-$(date +%F).tgz -C /path/to/bind/dir .
```

### (Recommended) Logical database backups

Safer than raw volume copies, and you can do them while the stack is up.
PostgreSQL:

```bash
docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" > backups/postgres-$(date +%F).sql
```

MariaDB:

```bash
docker compose exec -T mariadb sh -lc 'mysqldump -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE"' > backups/mariadb-$(date +%F).sql
```

### Restore notes

* Restore volumes/dirs **before** `docker compose up`.
* For bind mounts, expand the archive back to the same host paths.
* For DB dumps, restore with `psql`/`mysql` as appropriate.

## Roadmap

Have a request? Open an issue with a concrete example of the desired output.
