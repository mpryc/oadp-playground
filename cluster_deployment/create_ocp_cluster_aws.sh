#!/bin/sh

# Based on:
#  https://github.com/kaovilai/dotfiles/blob/master/zsh/functions/openshift/aws/create-ocp-aws.zsh

# Usage:
#   ./create-ocp-aws.sh [help|gather|delete|no-delete] [arm64|amd64]

set -eu

COMMAND=${1:-help}
ARCHITECTURE=${2:-amd64}
ARCH_SUFFIX="$ARCHITECTURE"

# Use system-installed openshift-install if available
OPENSHIFT_INSTALL="${OPENSHIFT_INSTALL:-$(command -v openshift-install || echo openshift-install)}"

# Optional: set cluster name prefix, default to "mpryc"
CLUSTER_NAME_PREFIX="${CLUSTER_NAME_PREFIX:-mpryc}"

# Required env variables
AWS_REGION="${AWS_REGION:-eu-central-1}"
AWS_BASEDOMAIN="${AWS_BASEDOMAIN:-mg.dog8code.com}"
TODAY=$(date +%Y%m%d)
OCP_MANIFESTS_DIR="${OCP_MANIFESTS_DIR:-$HOME/ocp-install}"
OCP_CREATE_DIR="$OCP_MANIFESTS_DIR/$TODAY-aws-$ARCH_SUFFIX"
CLUSTER_NAME="${CLUSTER_NAME_PREFIX}-${TODAY}-${ARCH_SUFFIX}"  # Max 21 chars

# --- HELP ---
if [ "$COMMAND" = "help" ]; then
    echo "Usage: $0 [help|gather|delete|no-delete] [arm64|amd64]"
    echo "Create an OpenShift cluster on AWS"
    echo ""
    echo "Options:"
    echo "  help      Show this message"
    echo "  gather    Gather bootstrap logs"
    echo "  delete    Destroy existing cluster"
    echo "  no-delete Skip cluster destruction"
    echo ""
    echo "Environment Variables:"
    echo "  OPENSHIFT_INSTALL         (default: detected in PATH)"
    echo "  AWS_REGION                (default: eu-central-1)"
    echo "  AWS_BASEDOMAIN            (default: mg.dog8code.com)"
    echo "  CLUSTER_NAME_PREFIX       (default: mpryc)"
    echo "  OCP_MANIFESTS_DIR         (default: \$HOME/ocp-install)"
    exit 0
fi

if ! "$OPENSHIFT_INSTALL" version | grep -q "release architecture $ARCHITECTURE"; then
    echo "WARN: $ARCHITECTURE architecture not supported in release image"
    echo "      Ensure the installer supports this architecture"
    exit 1
fi

echo "INFO: Using architecture: $ARCHITECTURE"
echo "INFO: Using cluster name: $CLUSTER_NAME"

if [ "$COMMAND" = "gather" ]; then
    "$OPENSHIFT_INSTALL" gather bootstrap --dir "$OCP_CREATE_DIR"
    exit 0
fi

if [ "$COMMAND" != "no-delete" ]; then
    "$OPENSHIFT_INSTALL" destroy cluster --dir "$OCP_CREATE_DIR" || echo "No existing cluster"
    "$OPENSHIFT_INSTALL" destroy bootstrap --dir "$OCP_CREATE_DIR" || echo "No existing bootstrap"
    rm -rf "$OCP_CREATE_DIR" && echo "Removed existing install dir" || echo "No install dir to remove"
fi

if [ "$COMMAND" = "delete" ]; then
    exit 0
fi

mkdir -p "$OCP_CREATE_DIR"
cat > "$OCP_CREATE_DIR/install-config.yaml" <<EOF
additionalTrustBundlePolicy: Proxyonly
apiVersion: v1
baseDomain: $AWS_BASEDOMAIN
compute:
- architecture: $ARCHITECTURE
  hyperthreading: Enabled
  name: worker
  platform: {}
  replicas: 3
controlPlane:
  architecture: $ARCHITECTURE
  hyperthreading: Enabled
  name: master
  platform: {}
  replicas: 3
metadata:
  creationTimestamp: null
  name: $CLUSTER_NAME
networking:
  clusterNetwork:
  - cidr: 10.128.0.0/14
    hostPrefix: 23
  machineNetwork:
  - cidr: 10.0.0.0/16
  networkType: OVNKubernetes
  serviceNetwork:
  - 172.30.0.0/16
platform:
  aws:
    region: $AWS_REGION
publish: External
pullSecret: '$(cat ~/pull-secret.txt)'
sshKey: |
  $(cat ~/.ssh/id_rsa.pub)
EOF

echo "INFO: Created install-config.yaml"

"$OPENSHIFT_INSTALL" create manifests --dir "$OCP_CREATE_DIR"

OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE="${OPENSHIFT_INSTALL_RELEASE_IMAGE_OVERRIDE:-}" \
"$OPENSHIFT_INSTALL" create cluster --dir "$OCP_CREATE_DIR" --log-level=info || \
"$OPENSHIFT_INSTALL" gather bootstrap --dir "$OCP_CREATE_DIR"

