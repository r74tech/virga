# Virga Documentation

This directory contains the VitePress documentation for Virga

## Development

```bash
# Install dependencies
bun install

# Start development server
bun docs:dev

# Build for production
bun docs:build

# Preview production build
bun docs:preview
```

## PR Preview

Pull requests that modify documentation will automatically deploy a preview at:
```
https://r74tech.github.io/virga/pr-preview/pr-{PR_NUMBER}/
```

The preview link will be commented on the PR by the GitHub Actions bot.

## Deployment

Documentation is automatically deployed to GitHub Pages when changes are pushed to the main branch.

Production URL: https://r74tech.github.io/virga/
