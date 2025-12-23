#!/bin/bash
set -e

VM_NAME="ceph-dev"
USER_ID="dev-user"
DISPLAY_NAME="Dev User"

# Check if user exists
if multipass exec "$VM_NAME" -- sudo radosgw-admin user info --uid="$USER_ID" &> /dev/null; then
    echo "User '$USER_ID' already exists. Retrieving credentials..."
else
    echo "Creating user '$USER_ID'..."
    multipass exec "$VM_NAME" -- sudo radosgw-admin user create --uid="$USER_ID" --display-name="$DISPLAY_NAME" &> /dev/null
fi

echo "User setup complete. Credentials:"

# Dump user info directly to stdout (variable capture causes hangs in some environments)
echo "Credentials for '$USER_ID':"
multipass exec "$VM_NAME" -- sudo radosgw-admin user info --uid="$USER_ID"

# Get VM IP
VM_IP=$(multipass info "$VM_NAME" | grep IPv4 | awk '{print $2}')
echo "Endpoint: http://$VM_IP:7480"


# Get VM IP
VM_IP=$(multipass info "$VM_NAME" | grep IPv4 | awk '{print $2}')
echo "Endpoint: http://$VM_IP:7480"
