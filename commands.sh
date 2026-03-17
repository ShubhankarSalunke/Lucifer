#!/bin/bash

# 1. Pull the latest changes from the main repository
echo "Updating main repository..."
git pull origin main

# 2. Initialize and update all submodules recursively
# This handles nested submodules and clones them if they're missing
echo "Syncing and updating submodules..."
git submodule update --init --recursive

# 3. (Optional) Pull the latest changes within each submodule
# By default, submodules point to a specific commit. 
# Run this if you want the submodules to track the latest remote branch:
# git submodule update --remote --merge

echo "Done! Your environment is synchronized."