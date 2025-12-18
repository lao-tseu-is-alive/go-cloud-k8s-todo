#!/bin/bash

# scripts/setup-template.sh
# Run after editing pkg/version/version.go

set -euo pipefail

VERSION_FILE="pkg/version/version.go"

if [ ! -f "$VERSION_FILE" ]; then
  echo "Error: $VERSION_FILE not found. Please create it first."
  exit 1
fi

echo "Reading values from $VERSION_FILE..."

# Robust extraction: find line with VarName = "value" and extract inside quotes
extract() {
  local var_name="$1"
  grep -E "${var_name}\s*=\s*\"" "$VERSION_FILE" | sed -E 's/.*=\s*"([^"]+)".*/\1/'
}

APP_NAME_CAMEL=$(extract "AppNameCamel")
APP_NAME_SNAKE=$(extract "AppNameSnake")
APP_BINARY=$(extract "AppBinary")
APP=$(extract "App")
REPOSITORY=$(extract "Repository")

# Fallback checks
if [[ -z "$APP_NAME_CAMEL" || -z "$APP_NAME_SNAKE" || -z "$APP_BINARY" || -z "$APP" || -z "$REPOSITORY" ]]; then
  echo "Error: Failed to extract one or more values from $VERSION_FILE. Check the file format."
  echo "Make sure each variable is on its own line like: VarName = \"value\""
  exit 1
fi

echo "Detected values:"
echo "  AppNameCamel: $APP_NAME_CAMEL"
echo "  AppNameSnake: $APP_NAME_SNAKE"
echo "  AppBinary:    $APP_BINARY"
echo "  App:          $APP"
echo "  Repository:   $REPOSITORY"

read -p "Continue with these values? (y/n): " confirm
[[ "$confirm" =~ ^[yY]$ ]] || { echo "Aborted."; exit 0; }

# Step 1: Rename directories and files
echo "Renaming directories and files..."

OLD_CMD="cmd/goCloudK8sThingServer"
NEW_CMD="cmd/$APP_BINARY"
if [ -d "$OLD_CMD" ]; then
  git mv "$OLD_CMD" "$NEW_CMD" || echo "Failed to rename cmd (already done?)."
else
  echo "Info: $OLD_CMD not found – likely already renamed or not present."
fi

# Rename thing → new snake name in common paths
for base in "api/proto" "gen" "pkg"; do
  old_path="${base}/thing"
  new_path="${base}/$APP_NAME_SNAKE"
  if [ -d "$old_path" ]; then
    git mv "$old_path" "$new_path"
  fi
done

# Rename proto files: thing*.proto → template*.proto
if [ -d "api/proto/$APP_NAME_SNAKE/v1" ]; then
  for file in api/proto/"$APP_NAME_SNAKE"/v1/thing*.proto; do
    [ -f "$file" ] || continue
    new_file="${file/thing/$APP_NAME_SNAKE}"
    git mv "$file" "$new_file"
  done
fi

# Step 2: String replacements (safe delimiter |)
echo "Replacing strings in files..."

find . -type f \
  \( -name '*.go' -o -name '*.proto' -o -name '*.yaml' -o -name '*.yml' -o -name 'Dockerfile' -o -name 'Makefile' -o -name '*.md' -o -name '*.sh' -o -name '*.env*' -o -name '*.sql' \) \
  ! -path './.git/*' ! -path './gen/*' \
  -print0 | xargs -0 sed -i '' \
    -e "s|goCloudK8sThingServer|$APP_BINARY|g" \
    -e "s|goCloudK8sThing|$APP|g" \
    -e "s|Thing|$APP_NAME_CAMEL|g" \
    -e "s|thing|$APP_NAME_SNAKE|g"

# Note: macOS sed uses -i '', Linux uses -i (no backup). This works on both.

# Step 3: Update go.mod
if [ -f go.mod ]; then
  REPO_PATH=$(echo "$REPOSITORY" | sed -E 's|^https?://github\.com/||')
  sed -i '' "s|module .*|module $REPO_PATH|g" go.mod
  echo "Updated go.mod to module $REPO_PATH"
fi

# Step 4: Regenerate protobufs
echo "Regenerating protobuf code..."
if [ -f scripts/buf_generate.sh ]; then
  ./scripts/buf_generate.sh
elif command -v buf >/dev/null 2>&1; then
  buf generate
else
  echo "Warning: buf not found – please run 'buf generate' manually later."
fi

go mod tidy

# Step 5: Commit
read -p "Commit changes? (y/n): " commit
if [[ "$commit" =~ ^[yY]$ ]]; then
  git add .
  git commit -m "chore: apply template customization from version.go"
fi

echo ""
echo "Setup complete! Your project is now customized for '$APP_NAME_CAMEL'."
echo "Review changes, run tests, and push when ready."