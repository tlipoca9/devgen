#!/bin/bash
# Test both plugin types and verify consistent output

set -e

cd "$(dirname "$0")"

echo "=== DevGen Plugin Consistency Test ==="
echo ""

# Clean up
rm -f testdata/testdata_marked.go
rm -f ../plugin-goplugin/goplugin.so

# Always use go run to get latest code
DEVGEN="go run ../../../cmd/devgen"

echo "Using devgen: $DEVGEN"
echo ""

# Test 1: Source type
echo "=== Test 1: Source Plugin ==="
cp devgen-source.toml devgen.toml
rm -f testdata/testdata_marked.go
$DEVGEN ./testdata
echo "Generated content:"
cat testdata/testdata_marked.go
echo ""
SOURCE_OUTPUT=$(cat testdata/testdata_marked.go | grep -v "^// Code generated")

# Test 2: Go Plugin type
echo "=== Test 2: Go Plugin (.so) ==="
echo "Building plugin..."
go build -buildmode=plugin -o ../plugin-goplugin/goplugin.so ../plugin-goplugin
cp devgen-goplugin.toml devgen.toml
rm -f testdata/testdata_marked.go
$DEVGEN ./testdata
echo "Generated content:"
cat testdata/testdata_marked.go
echo ""
PLUGIN_OUTPUT=$(cat testdata/testdata_marked.go | grep -v "^// Code generated")

# Compare outputs
echo "=== Comparing Outputs ==="
if [ "$SOURCE_OUTPUT" = "$PLUGIN_OUTPUT" ]; then
    echo "✅ Both plugin types produce consistent output!"
else
    echo "❌ Output mismatch detected!"
    echo ""
    echo "Source output:"
    echo "$SOURCE_OUTPUT"
    echo ""
    echo "Plugin output:"
    echo "$PLUGIN_OUTPUT"
    exit 1
fi

# Cleanup
rm -f devgen.toml
echo ""
echo "=== Test Complete ==="
