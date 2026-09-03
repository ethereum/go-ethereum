# Welcome to Default Team

This is your first page. Edit it in the Scalar editor or in your code editor — changes commit straight to the repository.

## Add another page

Create a new `.md` file under `docs/` and add it to your navigation group in `scalar.config.json` (under `navigation.routes` → group → `children`). For example:

```json
"/install": {
  "type": "page",
  "title": "Install",
  "filepath": "docs/install.md"
}
```

## Reference an OpenAPI document

Drop your API Document into the repo and add an entry like:

```json
"/api": {
  "type": "openapi",
  "title": "API Reference",
  "url": "./openapi.yaml"
}
```
