#!/usr/bin/env npx tsx

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { Command } from 'commander';
import YAML from 'yaml';
import { $ } from 'zx';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const kDefaultCephUser = 'hercules';

const kRepositoryRoot = path.join(__dirname, '..');
const kDockerComposeFile = path.join(__dirname, 'docker-compose.yml');
const kConfigFile = path.join(kRepositoryRoot, 'config.yaml');
const kExampleConfigFile = path.join(kRepositoryRoot, 'config.example.yaml');
const kSqlcConfigFile = path.join(kRepositoryRoot, 'sqlc.yaml');

process.chdir(__dirname);
process.env.PATH = `${__dirname}/bin:${process.env.PATH}`;

async function _readConfig() {
    if (!fs.existsSync(kConfigFile)) {
        throw new Error(`Config file not found at ${kConfigFile}`);
    }

    const configContent = fs.readFileSync(kConfigFile, 'utf-8');
    const document = YAML.parseDocument(configContent);

    return document;
}

type ConfigDocument = Awaited<ReturnType<typeof _readConfig>>;

async function _getMetaStoreDsn(document: ConfigDocument) {
    const config = document.toJS();

    if (!config.meta_store) {
        console.error("meta_store configuration not found in config.yaml");
        process.exit(1);
    }

    const { user, password, host, port, dbname, sslmode } = config.meta_store;
    const u = encodeURIComponent(user);
    const p = encodeURIComponent(password);

    return `postgres://${u}:${p}@${host}:${port}/${dbname}?sslmode=${sslmode}`;
}

// #region Docker

async function up(options: { verbose: boolean }) {
    if (options.verbose) $.verbose = true;
    else $.verbose = false;

    await $`docker compose -f ${kDockerComposeFile} up -d --build`;

    let ready = false;
    for (let i = 0; i < 60; i++) {
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
        await objectStoreCredentials({ user: kDefaultCephUser });
    }
}

async function down({ volumes }: { volumes: boolean }) {
    await $`docker compose -f ${kDockerComposeFile} down ${volumes ? '-v' : ''}`;
}

async function status() {
    await $`docker compose -f ${kDockerComposeFile} ps`;
}

// #endregion

// #region Config

async function getConfig(this: Command, path: string): Promise<void> {
    const document = await _readConfig();
    const value = document.getIn(path.split('.'));
    console.log(JSON.stringify(value));
}

// #endregion

// #region Object Store

async function objectStoreCredentials({ user }: { user: string }) {
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
            const health = await $`ceph -s`.nothrow();
            if (health.stdout.includes("health: HEALTH_OK")) break;
        } catch { }
        await new Promise(resolve => setTimeout(resolve, 5000));
        process.stdout.write(".");
    }

    while (true) {
        try {
            const status = await $`microceph status`.nothrow();
            if (status.stdout.includes("rgw")) break;
        } catch { }
        await new Promise(resolve => setTimeout(resolve, 2000));
    }

    let rawUserInfo = "";
    const create = await $`radosgw-admin user create --uid="${user}" --display-name="${user}"`.nothrow();
    if (create.exitCode === 0) {
        rawUserInfo = create.stdout;
    } else {
        const info = await $`radosgw-admin user info --uid=${user}`;
        rawUserInfo = info.stdout;
    }

    let accessKey = "";
    let secretKey = "";
    try {
        const userInfo = JSON.parse(rawUserInfo);
        if (userInfo.keys && userInfo.keys.length > 0) {
            accessKey = userInfo.keys[0].access_key;
            secretKey = userInfo.keys[0].secret_key;
        }
    } catch (e) {
        console.error("Failed to parse credentials logic:", e);
        process.exit(1);
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

    doc.setIn(['object_store', 'access_key'], accessKey);
    doc.setIn(['object_store', 'secret_key'], secretKey);

    fs.writeFileSync(kConfigFile, doc.toString());
}

// #endregion

// #region Meta Store

async function metaStoreGetDsn(this: Command, ...args: any[]): Promise<void> {
    const document = await _readConfig();
    console.log(await _getMetaStoreDsn(document));
}

async function metaStoreMigrate() {
    const document = await _readConfig();
    process.env.DATABASE_URL = await _getMetaStoreDsn(document);

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

// #endregion

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

    // Config

    const configCmd = program.command('config')
        .description('Manage configuration');

    configCmd.command('get')
        .description('Get configuration value by attribute path')
        .argument('<path>', 'Attribute path')
        .action(getConfig);

    // Object Store

    const objectStoreCmd = program.command('objectstore')
        .description('Manage object store')

    objectStoreCmd.command('credentials')
        .description('Generate/Refresh S3 credentials')
        .requiredOption('-u, --user <user>', 'User name')
        .action(objectStoreCredentials);

    // Metadata Store

    const metaStoreCmd = program.command('metastore')
        .description('Manage metadata store')

    metaStoreCmd.command('get-dsn')
        .description('Get database connection string')
        .action(metaStoreGetDsn);

    metaStoreCmd.command('migrate')
        .description('Run database migrations')
        .action(metaStoreMigrate);

    metaStoreCmd.command('generate')
        .description('Generate database migrations')
        .action(metaStoreGenerate);

    program.parse(process.argv);
}

main();
