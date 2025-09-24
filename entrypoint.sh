#!/bin/sh

# Handle base64-encoded private key for container deployments
if [ -n "$HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY_B64" ]; then
    echo "Decoding base64 private key..."
    echo "$HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY_B64" | base64 -d > /tmp/github-key.pem
    chmod 600 /tmp/github-key.pem
    export HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY=/tmp/github-key.pem
fi

# Run the actual command
exec "$@"