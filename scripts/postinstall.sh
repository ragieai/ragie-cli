#!/bin/bash

# Create symlink if it doesn't exist
if [ ! -e /usr/local/bin/ragie ]; then
  ln -s /usr/bin/ragie /usr/local/bin/ragie 2>/dev/null || true
fi

# Set permissions
chmod 755 /usr/bin/ragie

# Print success message
echo "Ragie has been successfully installed!"
echo "Run 'ragie --help' to get started."

exit 0 