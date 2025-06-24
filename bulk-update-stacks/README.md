# Spacelift Stack Update Script

A Python script to bulk update all Spacelift stacks in your environment with standardized configurations.

## What It Does

This script:

1. Updates each stack with standardized settings while preserving stack-specific data
2. Provides logging for success/failure tracking
3. Continues processing even if individual stacks fail

## Setup

1. Install dependencies:

   ```bash
   pip install -r requirements.txt
   ```

2. Update the `CONFIG` dictionary with whatever settings you'd like to update across all stacks

## Usage

   ```bash
   python spacelift_update.py
   ```

## Customization

Modify the `variables` dictionary to change what gets updated on each stack.

## Security

- API tokens handled securely through separate auth module
- No sensitive data logged
- Minimal required permissions recommended

## Limitations

- This script only works for Terraform/OpenTofu stacks
