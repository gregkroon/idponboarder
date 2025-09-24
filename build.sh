#!/usr/bin/env bash
set -euo pipefail

# --- Defaults (override via env) ---
: "${DEFAULT_NS:=munkys123}"                       # used only if you pass a bare image name
: "${PLATFORMS:=linux/amd64,linux/arm64}"         # target platforms
: "${BUILDER_NAME:=multiplatform}"                 # buildx builder name

# --- Args ---
IMAGE_INPUT="${1:-harness-onboarder}"              # can be bare ('onboarder') or 'ns/name' or 'reg/ns/name'
TAG="${2:-latest}"

# If IMAGE_INPUT contains a slash (namespace/registry present), use as-is.
# Otherwise, prefix with DEFAULT_NS.
if [[ "$IMAGE_INPUT" == *"/"* ]]; then
  IMAGE_NO_TAG="$IMAGE_INPUT"
else
  IMAGE_NO_TAG="${DEFAULT_NS}/${IMAGE_INPUT}"
fi

IMAGE_REF="${IMAGE_NO_TAG}:${TAG}"

echo "Building multi-platform image: ${IMAGE_REF}"
echo "Platforms: ${PLATFORMS}"

# --- Ensure buildx is available ---
if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker not found in PATH" >&2
  exit 1
fi

if ! docker buildx version >/dev/null 2>&1; then
  echo "ERROR: docker buildx is not available. Install/enable Buildx first." >&2
  exit 1
fi

# --- Ensure/Use a named builder (so cache persists between runs) ---
if ! docker buildx inspect "$BUILDER_NAME" >/dev/null 2>&1; then
  echo "Creating buildx builder: $BUILDER_NAME"
  docker buildx create --name "$BUILDER_NAME" --use
else
  docker buildx use "$BUILDER_NAME"
fi

# Optional: boot the builder (esp. on first use / remote drivers)
docker buildx inspect --bootstrap >/dev/null

# --- Build & push ---
docker buildx build \
  --platform "$PLATFORMS" \
  --tag "$IMAGE_REF" \
  --push \
  .

echo "✅ Multi-platform build complete: ${IMAGE_REF}"

