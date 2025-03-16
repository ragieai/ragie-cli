#!/bin/bash

# Remove symlink if it exists
if [ -L /usr/local/bin/ragie ]; then
  rm -f /usr/local/bin/ragie
fi

echo "Ragie is being removed..."

exit 0 