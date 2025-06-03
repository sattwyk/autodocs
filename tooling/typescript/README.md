# @autodocs/typescript-config

Shared TypeScript configurations for the autodocs monorepo.

## Available Configurations

### Base Configuration (`@autodocs/typescript-config/base`)

The foundational TypeScript configuration with strict settings and modern features.

- **Target**: ES2022
- **Module**: ESNext
- **Strict mode**: Enabled
- **Additional checks**: `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`, `noImplicitReturns`, etc.

### Next.js Configuration (`@autodocs/typescript-config/nextjs`)

Extends the base config with Next.js specific settings.

- **Target**: ES2017 (Next.js compatibility)
- **JSX**: preserve
- **DOM types**: Included
- **Next.js plugin**: Enabled

### Node.js Configuration (`@autodocs/typescript-config/node`)

Extends the base config for Node.js applications.

- **Target**: ES2022
- **Module Resolution**: Node
- **Node types**: Included

### Library Configuration (`@autodocs/typescript-config/library`)

Extends the base config for packages that need to be built for distribution.

- **Target**: ES2020
- **Declaration files**: Generated
- **Source maps**: Enabled
- **Output directory**: `./dist`

## Usage

In your `tsconfig.json`:

```json
{
  "extends": "@autodocs/typescript-config/nextjs",
  "compilerOptions": {
    "paths": {
      "@/*": ["./*"]
    }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

## Package Dependencies

To use these configurations, add the package as a dev dependency:

```json
{
  "devDependencies": {
    "@autodocs/typescript-config": "workspace:*",
    "typescript": "^5"
  }
}
```
