#!/bin/bash
# Clones the published A2A gRPC spec, renames the proto file to a2av0.proto,
# and generates Go types in ./a2apb.
#
# The rename is intentional: the protobuf file descriptor registers as
# "a2av0.proto" instead of "a2a.proto", avoiding a name clash with other
# protocol versions (e.g. a2a-go/v2/a2apb/v1).
#
# Ensure $GOBIN is in path and dependencies are installed:
# > go install github.com/bufbuild/buf/cmd/buf@latest
# > go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# > go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#
# Then run:
# > ./tools/generate_types.sh

set -euo pipefail

REPO_URL="https://github.com/a2aproject/A2A.git"
COMMIT="e7cf203ecdd7003c0f1740bb712248d2b5252bf1"
PROTO_SUBDIR="specification/grpc"
OUTPUT_DIR="./a2apb"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Cloning A2A spec at ${COMMIT:0:12}..."
git clone --no-checkout --quiet "$REPO_URL" "$TMPDIR/A2A"
git -C "$TMPDIR/A2A" checkout --quiet "$COMMIT" -- "$PROTO_SUBDIR"

echo "Renaming a2a.proto -> a2av0.proto..."
mv "$TMPDIR/A2A/$PROTO_SUBDIR/a2a.proto" "$TMPDIR/A2A/$PROTO_SUBDIR/a2av0.proto"

cat > "$TMPDIR/buf.gen.yaml" <<EOF
version: v2
inputs:
  - directory: $TMPDIR/A2A/$PROTO_SUBDIR

managed:
  enabled: true
  override:
    - file_option: go_package
      path: a2av0.proto
      value: github.com/a2aproject/a2a-go/a2apb

plugins:
  - remote: buf.build/protocolbuffers/go
    out: $OUTPUT_DIR
    opt:
      - paths=source_relative

  - remote: buf.build/grpc/go
    out: $OUTPUT_DIR
    opt:
      - paths=source_relative
EOF

echo "Generating Go code..."
buf generate --template "$TMPDIR/buf.gen.yaml"

echo "Done: $OUTPUT_DIR/a2av0.pb.go, $OUTPUT_DIR/a2av0_grpc.pb.go"
