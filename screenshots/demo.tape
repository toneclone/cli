# VHS Tape for ToneClone CLI Demo
# https://github.com/charmbracelet/vhs

Output demo.gif

# Terminal settings
Set FontSize 16
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 50ms
Set PlaybackSpeed 1.0

# Set up function for git to use sample file (for demo purposes)
Hide
Type "git() { cat sample.diff; }"
Enter
Sleep 500ms
Ctrl+L
Show

# Show help command
Type "toneclone --help"
Enter
Sleep 3s

# Clear screen
Ctrl+L
Sleep 500ms

# Write command - commit message with piped diff
Type "git diff | toneclone write --persona='Technical Writer' --knowledge='Git Commits'"
Enter
Sleep 10s

# Clear screen
Ctrl+L
Sleep 500ms

# Write command - marketing copy
Type "toneclone write --persona='Product Marketer' --knowledge='Product Launch' --prompt='Announce new CLI tool to write with AI in your unique voice and style'"
Enter
Sleep 8s

# Clear screen
Ctrl+L
Sleep 500ms

# List personas
Type "toneclone personas list"
Enter
Sleep 3s

# Clear screen
Ctrl+L
Sleep 500ms

# List knowledge cards
Type "toneclone knowledge list"
Enter
Sleep 3s

# Final pause
Sleep 2s
