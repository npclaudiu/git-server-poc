#!/bin/bash
set -e

VM_NAME="devenv"

echo "Installing MicroCeph on '$VM_NAME'..."

# Install microceph
multipass exec "$VM_NAME" -- sudo snap install microceph --channel=latest/edge

# Bootstrap cluster
echo "Bootstrapping MicroCeph cluster..."
if multipass exec "$VM_NAME" -- sudo microceph status | grep -q "Services:.*mon"; then
    echo "Cluster already bootstrapped."
else
    multipass exec "$VM_NAME" -- sudo microceph cluster bootstrap
fi

# Add OSDs
echo "Adding loopback disks for OSDs..."
# Check if we have at least 3 Disks
DISK_COUNT=$(multipass exec "$VM_NAME" -- sudo microceph status | grep "Disks:" | awk '{print $2}' | tr -d '\r')
if [ -z "$DISK_COUNT" ]; then DISK_COUNT=0; fi

if [ "$DISK_COUNT" -lt 3 ]; then
    multipass exec "$VM_NAME" -- sudo microceph disk add loop,4G,3
else
    echo "OSDs already added (found $DISK_COUNT)."
fi

# Enable RGW
echo "Enabling RGW..."
if multipass exec "$VM_NAME" -- sudo microceph status | grep -q "Services:.*rgw"; then
    echo "RGW already enabled."
else
    multipass exec "$VM_NAME" -- sudo microceph enable rgw
fi

echo "MicroCeph setup complete."
