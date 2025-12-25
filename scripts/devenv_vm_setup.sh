#!/bin/bash
set -e

VM_NAME="ceph-dev"
CPUS="4"
MEMORY="8G"
DISK="20G"

# Check if multipass is installed
if ! command -v multipass &> /dev/null; then
    echo "Error: multipass is not installed. Please install it first (brew install --cask multipass for macOS)."
    exit 1
fi

# Check if VM already exists
if multipass list | grep -q "$VM_NAME"; then
    echo "VM '$VM_NAME' already exists."
    # Check if it's running
    if ! multipass info "$VM_NAME" | grep -q "State:.*Running"; then
        echo "Starting VM '$VM_NAME'..."
        multipass start "$VM_NAME"
    fi
else
    echo "Creating VM '$VM_NAME' with $CPUS CPUs, $MEMORY RAM, and $DISK disk..."
    multipass launch jammy --name "$VM_NAME" --cpus "$CPUS" --memory "$MEMORY" --disk "$DISK"
fi

echo "VM '$VM_NAME' is ready."
