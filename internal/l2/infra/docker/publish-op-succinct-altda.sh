#!/usr/bin/env bash
set -euo pipefail

# Publish the op-succinct AltDA validity proposer image to Docker Hub.
#
# Usage:
#   DOCKERHUB_TOKEN=dckr_pat_... ./publish-op-succinct-altda.sh [TAG]
#
# TAG defaults to the current git short SHA.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
OP_SUCCINCT_PATH="$REPO_ROOT/op-succinct-ssv"

DOCKERHUB_USER="ayaz461"
IMAGE_NAME="$DOCKERHUB_USER/op-succinct-altda"
TAG="${1:-$(git -C "$OP_SUCCINCT_PATH" rev-parse --short HEAD)}"

if [ -z "${DOCKERHUB_TOKEN:-}" ]; then
  echo "Error: DOCKERHUB_TOKEN is not set." >&2
  echo "Export it before running: export DOCKERHUB_TOKEN=dckr_pat_..." >&2
  exit 1
fi

echo "Logging in to Docker Hub as $DOCKERHUB_USER..."
echo "$DOCKERHUB_TOKEN" | docker login -u "$DOCKERHUB_USER" --password-stdin

echo "Building $IMAGE_NAME:$TAG (linux/amd64) from op-succinct.altda.Dockerfile..."
docker buildx build \
  --platform linux/amd64 \
  -f "$SCRIPT_DIR/op-succinct.altda.Dockerfile" \
  --build-arg CARGO_PROFILE=release \
  -t "$IMAGE_NAME:$TAG" \
  -t "$IMAGE_NAME:latest" \
  --push \
  "$OP_SUCCINCT_PATH"

echo "Done. Published:"
echo "  $IMAGE_NAME:$TAG"
echo "  $IMAGE_NAME:latest"
