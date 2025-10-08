#!/bin/bash

# ToneClone CLI Demo Setup Script
# This script sets up the demo account with sample personas, knowledge cards, and training data

set -e

echo "🚀 Setting up ToneClone CLI demo account..."

# API key for demo account
DEMO_API_KEY="tc_live_6GKKFXTXIF3O5U7WFLTNKDIALSUQKGDMB6YD2HYRAA3L56IAWR4Q"

# Path to writing samples
SAMPLES_DIR="$HOME/projects/writing-samples/jon/emails"

echo ""
echo "📝 Step 1: Authenticating with demo account..."
echo "$DEMO_API_KEY" | toneclone auth login --name="demo" --from-stdin --force

echo ""
echo "👤 Step 2: Creating personas..."
toneclone personas create --name="Developer"
toneclone personas create --name="Product Marketer"
toneclone personas create --name="Technical Writer"

echo ""
echo "📚 Step 3: Creating knowledge cards..."
toneclone knowledge create --name="Git Commits" --instructions="Write detailed git commit messages using this structure:

<type>(<scope>): <short summary>

<body>
- Explain the motivation for the change
- Describe what was changed and how
- Note any breaking changes or important side effects

<footer>
- Reference related issues (e.g., Fixes #123, Closes #456)
- Note breaking changes with 'BREAKING CHANGE:'
- Add co-authors if applicable

Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore
Keep summary under 72 chars. Wrap body at 72 chars.
Be specific and technical - explain WHY not just WHAT."
toneclone knowledge create --name="Product Launch" --instructions="Write compelling marketing copy for product announcements. Be concise, highlight key benefits, and create excitement about the new features or product.

Company: ToneClone
Website: https://toneclone.ai

Include the company name and website link naturally in the announcement."
toneclone knowledge create --name="Release Notes" --instructions="Write clear, structured release notes and changelogs. Organize by category (Features, Fixes, Improvements), use bullet points, and be specific about what changed."

echo ""
echo "📄 Step 4: Uploading training data..."
toneclone training add --file="$SAMPLES_DIR/2025-01-24_Follow_Up_and_Feedback.txt" --persona="Developer"
toneclone training add --file="$SAMPLES_DIR/2025-02-17_Can_we_chat_about_ToneClone.txt" --persona="Developer"
toneclone training add --file="$SAMPLES_DIR/2025-03-27_Checking_in.txt" --persona="Developer"
toneclone training add --file="$SAMPLES_DIR/2025-02-24_Thanks_for_the_chat.txt" --persona="Developer"

echo ""
echo "✅ Demo account setup complete!"
echo ""
echo "You can now run: cd screenshots && vhs demo.tape"
