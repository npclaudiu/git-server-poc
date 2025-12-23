#!/bin/bash
set -e

VM_NAME="ceph-dev"
DB_USER="git_user"
DB_PASS="git_password"
DB_NAME="git_server"

echo "Setting up PostgreSQL on '$VM_NAME'..."

# Install PostgreSQL
if ! multipass exec "$VM_NAME" -- dpkg -l | grep -q postgresql; then
    echo "Installing PostgreSQL..."
    multipass exec "$VM_NAME" -- sudo apt-get update
    multipass exec "$VM_NAME" -- sudo apt-get install -y postgresql postgresql-common
else
    echo "PostgreSQL already installed."
fi

# Configure Network Access
echo "Configuring remote access..."
# Locate config files dynamically
PG_CONF=$(multipass exec "$VM_NAME" -- find /etc/postgresql -name postgresql.conf | head -n 1)
PG_HBA=$(multipass exec "$VM_NAME" -- find /etc/postgresql -name pg_hba.conf | head -n 1)

if [ -z "$PG_CONF" ] || [ -z "$PG_HBA" ]; then
    echo "Error: Could not locate PostgreSQL configuration files."
    exit 1
fi

echo "Found config: $PG_CONF"
echo "Found hba: $PG_HBA"

# 1. listen_addresses = '*'
multipass exec "$VM_NAME" -- sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" "$PG_CONF"

# 2. Allow remote connections in pg_hba.conf
# We use a temp file to avoid complex escaping in sed/echo over ssh
cat <<EOF > /tmp/pg_hba_patch
host    all             all             0.0.0.0/0               scram-sha-256
EOF
multipass transfer /tmp/pg_hba_patch "$VM_NAME":/tmp/pg_hba_patch
rm /tmp/pg_hba_patch

# Append if not already present
multipass exec "$VM_NAME" -- bash -c "sudo grep -q '0.0.0.0/0' '$PG_HBA' || cat /tmp/pg_hba_patch | sudo tee -a '$PG_HBA' > /dev/null"
multipass exec "$VM_NAME" -- rm /tmp/pg_hba_patch

# Restart PostgreSQL to apply changes
multipass exec "$VM_NAME" -- sudo systemctl restart postgresql

# set user and database
echo "Configuring database and user..."

# Create User
if ! multipass exec "$VM_NAME" -- sudo -i -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'" | grep -q 1; then
    multipass exec "$VM_NAME" -- sudo -i -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASS';"
    echo "User '$DB_USER' created."
else
    echo "User '$DB_USER' already exists."
fi

# Create Database
if ! multipass exec "$VM_NAME" -- sudo -i -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -q 1; then
    multipass exec "$VM_NAME" -- sudo -i -u postgres psql -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"
    echo "Database '$DB_NAME' created."
else
    echo "Database '$DB_NAME' already exists."
fi

# Get Connection Info
VM_IP=$(multipass info "$VM_NAME" | grep IPv4 | awk '{print $2}')
echo "PostgreSQL setup complete."
echo "Connection String: postgres://$DB_USER:$DB_PASS@$VM_IP:5432/$DB_NAME"
