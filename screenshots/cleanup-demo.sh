#!/bin/bash

# ToneClone CLI Demo Cleanup Script
# This script removes all demo data from the account

set -e

echo "🧹 Cleaning up ToneClone CLI demo account..."

echo ""
echo "Getting persona and knowledge card IDs..."

# Get all personas and delete user-created ones
PERSONAS=$(toneclone personas list --format=json | jq -r '.[] | select(.source == "User") | .id')
for persona_id in $PERSONAS; do
    echo "🗑️  Deleting persona: $persona_id"
    toneclone personas delete "$persona_id" --confirm 2>/dev/null || true
done

# Get all knowledge cards and delete them
KNOWLEDGE=$(toneclone knowledge list --format=json | jq -r '.[].id')
for knowledge_id in $KNOWLEDGE; do
    echo "🗑️  Deleting knowledge card: $knowledge_id"
    toneclone knowledge delete "$knowledge_id" --confirm 2>/dev/null || true
done

# Get all training files and delete them
TRAINING=$(toneclone training list --format=json | jq -r '.[].id')
for file_id in $TRAINING; do
    echo "🗑️  Deleting training file: $file_id"
    toneclone training remove --file-id="$file_id" --confirm 2>/dev/null || true
done

echo ""
echo "✅ Demo account cleanup complete!"
echo ""
echo "Note: The demo profile is still configured in your ~/.toneclone.yaml"
echo "To remove it completely, run: toneclone auth logout"
