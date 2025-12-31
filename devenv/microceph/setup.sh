#!/bin/bash
set -e

# Log all output to a file for debugging
exec > >(tee -a /var/log/microceph-setup.log) 2>&1

echo "Starting MicroCeph setup..."

# Wait for snapd
until snap list > /dev/null 2>&1; do
  echo "Waiting for snapd to be ready..."
  sleep 2
done

# Install Core Snaps
# Always try to install to be safe compared to 'snap list'
echo "Ensuring core snaps are installed..."
snap install core24 || true
snap install core22 || true
snap install core || true

# Install MicroCeph
if ! snap list microceph > /dev/null 2>&1; then
    echo "Installing microceph..."
    snap install microceph
    
    # Wait for service
    until snap services microceph | grep -q "active"; do
        echo "Waiting for microceph service..."
        sleep 2
    done
fi

# Bootstrap Cluster
if ! microceph status | grep -q "health: HEALTH_OK"; then
    if ! microceph status | grep -q "Services: mds, mgr, mon"; then
        echo "Bootstrapping cluster..."
        microceph cluster bootstrap
    fi
fi

# Add OSDs
echo "Checking OSDs..."
DISK_COUNT=$(microceph status | grep "Disks:" | awk '{print $2}' | tr -d '\r')
if [ -z "$DISK_COUNT" ]; then DISK_COUNT=0; fi

if [ "$DISK_COUNT" -lt 3 ]; then
    echo "Adding loopback disks for OSDs..."
    microceph disk add loop,4G,3
fi

# Enable RGW
if ! microceph status | grep -q "Services:.*rgw"; then
    echo "Enabling RGW..."
    microceph enable rgw
fi

echo "MicroCeph setup complete."
