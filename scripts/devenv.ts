#!/usr/bin/env npx tsx

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { Command } from 'commander';
import { $ } from 'zx';
import YAML from 'yaml';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const kRepositoryRoot = path.join(__dirname, '..');

const kDockerComposeFile = path.join(__dirname, 'docker-compose.yml');
const kConfigFile = path.join(kRepositoryRoot, 'config.yaml');
const kExampleConfigFile = path.join(kRepositoryRoot, 'config.example.yaml');
const kSqlcConfigFile = path.join(kRepositoryRoot, 'sqlc.yaml');

process.chdir(__dirname);

async function up(options: { verbose: boolean }) {
    if (options.verbose) $.verbose = true;
    else $.verbose = false;

    await $`docker compose -f ${kDockerComposeFile} up -d --build`;

    let ready = false;
    for (let i = 0; i < 30; i++) {
        await new Promise(resolve => setTimeout(resolve, 5_000));
        process.stdout.write(".");
        try {
            const status = await $`docker exec microceph /snap/bin/microceph.ceph -s`.nothrow();
            if (status.exitCode === 0 && status.stdout.includes("health: HEALTH_OK")) {
                ready = true;
                break;
            }
        } catch { }
    }

    if (!ready) {
        console.error("MicroCeph failed to become ready. Check logs: `docker logs microceph`.");
    } else {
        await objectStoreCredentials();
    }
}

async function down({ volumes }: { volumes: boolean }) {
    await $`docker compose -f ${kDockerComposeFile} down ${volumes ? '-v' : ''}`;
}

async function status() {
    await $`docker compose -f ${kDockerComposeFile} ps`;
}

async function objectStoreCredentials() {
    while (true) {
        try {
            const ps = await $`docker ps --filter "name=microceph" --filter "status=running"`;
            if (ps.stdout.includes("microceph")) break;
        } catch { }
        await new Promise(resolve => setTimeout(resolve, 2000));
        process.stdout.write(".");
    }

    while (true) {
        try {
            const health = await $`docker exec microceph /snap/bin/microceph.ceph -s`.nothrow();
            if (health.stdout.includes("health: HEALTH_OK")) break;
        } catch { }
        await new Promise(resolve => setTimeout(resolve, 5000));
        process.stdout.write(".");
    }

    while (true) {
        try {
            const status = await $`docker exec microceph /snap/bin/microceph status`.nothrow();
            if (status.stdout.includes("rgw")) break;
        } catch { }
        await new Promise(resolve => setTimeout(resolve, 2000));
    }

    let credsJson = "";
    const create = await $`docker exec microceph /snap/bin/microceph.radosgw-admin user create --uid="dev" --display-name="Developer"`.nothrow();
    if (create.exitCode === 0) {
        credsJson = create.stdout;
    } else {
        const info = await $`docker exec microceph /snap/bin/microceph.radosgw-admin user info --uid=dev`;
        credsJson = info.stdout;
    }

    let accessKey = "";
    let secretKey = "";
    try {
        const data = JSON.parse(credsJson);
        if (data.keys && data.keys.length > 0) {
            accessKey = data.keys[0].access_key;
            secretKey = data.keys[0].secret_key;
        }
    } catch (e) {
        console.error("Failed to parse credentials logic:", e);
    }

    if (!accessKey || !secretKey) {
        const akMatch = credsJson.match(/"access_key": "([^"]+)"/);
        const skMatch = credsJson.match(/"secret_key": "([^"]+)"/);
        if (akMatch) accessKey = akMatch[1];
        if (skMatch) secretKey = skMatch[1];
    }

    if (!fs.existsSync(kConfigFile)) {
        if (fs.existsSync(kExampleConfigFile)) {
            fs.copyFileSync(kExampleConfigFile, kConfigFile);
        } else {
            fs.writeFileSync(kConfigFile, "{}");
        }
    }

    const configContent = fs.readFileSync(kConfigFile, 'utf-8');

    const doc = YAML.parseDocument(configContent);

    const newObjectStore = {
        endpoint: "http://localhost:8000",
        access_key: accessKey,
        secret_key: secretKey,
        bucket: "git-lfs",
        region: "us-east-1"
    };

    doc.set('object_store', newObjectStore);
    fs.writeFileSync(kConfigFile, doc.toString());
}

async function metaStoreMigrate() {
    console.log("Running database migrations...");

    if (!fs.existsSync(kConfigFile)) {
        console.error(`Config file not found at ${kConfigFile}`);
        process.exit(1);
    }

    const configContent = fs.readFileSync(kConfigFile, 'utf-8');
    const config = YAML.parse(configContent);

    if (!config.meta_store) {
        console.error("meta_store configuration not found in config.yaml");
        process.exit(1);
    }

    const { user, password, host, port, dbname, sslmode } = config.meta_store;
    process.env.DATABASE_URL = `postgres://${user}:${password}@${host}:${port}/${dbname}?sslmode=${sslmode}`;

    const metastoreDir = path.join(kRepositoryRoot, 'internal', 'metastore', 'pg');
    const migrationsDir = path.join(metastoreDir, 'migrations');
    const schemaFile = path.join(metastoreDir, 'schema.sql');

    await $`dbmate -d ${migrationsDir} -s ${schemaFile} up`;

    const schema = await fs.readFileSync(schemaFile, 'utf-8');
    const regex = /\\restrict|\\unrestrict/g;
    const newSchema = schema.replace(regex, '-- $&');
    fs.writeFileSync(schemaFile, newSchema);
}

async function metaStoreGenerate() {
    await $`sqlc generate -f ${kSqlcConfigFile}`;
}

async function main() {
    const program = new Command();

    program
        .name('devenv')
        .description('Manage the Docker-based development environment')
        .version('1.0.0');

    // Docker environment

    program.command('up')
        .description('Start the environment and configure MicroCeph')
        .option('-v, --verbose', 'Run with verbose logging')
        .action(up);

    program.command('down')
        .description('Stop the environment and remove volumes')
        .option('-v, --volumes', 'Remove volumes', true)
        .action(down);

    program.command('status')
        .description('Show status of containers')
        .action(status);

    // Object Store

    const objectStoreCmd = program.command('objectstore')
        .description('Manage object store')

    objectStoreCmd.command('credentials')
        .description('Generate/Refresh S3 credentials')
        .action(objectStoreCredentials);

    // Metadata Store

    const metaStoreCmd = program.command('metastore')
        .description('Manage metadata store')

    metaStoreCmd.command('migrate')
        .description('Run database migrations')
        .action(metaStoreMigrate);

    metaStoreCmd.command('generate')
        .description('Generate database migrations')
        .action(metaStoreGenerate);

    program.parse(process.argv);
}

main();
