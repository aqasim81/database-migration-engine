# Changelog

All notable changes to this project. Generated from conventional commits by git-cliff.

## [Unreleased]

### Features

- Scaffold project structure with CLI skeleton and config loading ([cf6b80b](https://github.com/aqasim81/database-migration-engine/commit/cf6b80bea3183e1fcaf314fa09813cc33aea7fa4))
- Add SQL parser wrapper and migration file loading ([cb67c64](https://github.com/aqasim81/database-migration-engine/commit/cb67c6439491bb96df8df16845c0d558a70cdb9c))
- Implement danger analysis engine with 9 detection rules ([e65700f](https://github.com/aqasim81/database-migration-engine/commit/e65700f7d0894aeb88c44e420f6c83089c6bd758))
- Add database connection pool with advisory locks (phase 4) ([44bc720](https://github.com/aqasim81/database-migration-engine/commit/44bc720a8f145e774b31d644c5b006b652ef41db))
- Add migration tracker with schema_migrations CRUD (phase 4) ([9786b5c](https://github.com/aqasim81/database-migration-engine/commit/9786b5c74b8301c116063f4a0c660fb47003ee4e))
- Implement migration executor with transaction safety (phase 5) ([2e5b7b2](https://github.com/aqasim81/database-migration-engine/commit/2e5b7b2aa8bb3b26f2c54522756574167fc81783))
- Wire apply command with executor, analyzer, and progress output (phase 5) ([e255860](https://github.com/aqasim81/database-migration-engine/commit/e255860973f3b226adba235d1de0837ecf781e34))
- Extend MigrationTracker interface for rollback ([37180c2](https://github.com/aqasim81/database-migration-engine/commit/37180c28434bdf90cbf2fde83432f9e496f350a5))
- Implement rollback engine ([77c7b5b](https://github.com/aqasim81/database-migration-engine/commit/77c7b5bb65f3d14b91ffbe60d12c6769e13e13f3))
- Wire rollback CLI command ([b222457](https://github.com/aqasim81/database-migration-engine/commit/b222457e3171bb27aed143c84c83c589ff3fc3e5))
- Add impact estimation heuristics for migration planning ([1bc2d1f](https://github.com/aqasim81/database-migration-engine/commit/1bc2d1f04be0060f9f35d72f54088d235167078b))
- Add migration planner with execution plan builder ([4811243](https://github.com/aqasim81/database-migration-engine/commit/4811243f57d78c458fc171f2531bd52a4e438f0c))
- Wire plan CLI command with formatted table output ([10baace](https://github.com/aqasim81/database-migration-engine/commit/10baace5e1b3647caa2853eda407794988e96783))
- Add output formatting with lipgloss styles and JSON support ([a658a67](https://github.com/aqasim81/database-migration-engine/commit/a658a673984d47cb2aa0280db9df4a039d4f953c))
- Add format flag support to analyze command ([dae56d0](https://github.com/aqasim81/database-migration-engine/commit/dae56d004fd898ab70ee0fbdd4b630a170368228))
- Implement status command with applied/pending cross-reference ([7f0eccb](https://github.com/aqasim81/database-migration-engine/commit/7f0eccb9c379d6e29f3f4732a66cdc1920f95eb2))
- Add plan JSON output and apply confirmation prompts ([83da7b3](https://github.com/aqasim81/database-migration-engine/commit/83da7b3be938239d7f99f510c9f577ed6c820040))

### Bug fixes

- Add nil guards to all analyzer rules and helpers ([a45c074](https://github.com/aqasim81/database-migration-engine/commit/a45c07472f864e22605ad2ff8b7a19e8ef137b40))
- Use stable Go version in CI to match go.mod 1.25 ([5e67c3c](https://github.com/aqasim81/database-migration-engine/commit/5e67c3c5d31d0e4f7efd16d66bc4699535c5d203))
- Use goinstall mode for golangci-lint in CI ([46bbcb0](https://github.com/aqasim81/database-migration-engine/commit/46bbcb0bc2e82cc7695de201afb5358f257c08bf))
- Pin golangci-lint to v2 in CI to match config format ([d821f23](https://github.com/aqasim81/database-migration-engine/commit/d821f23d268a7a0919349b131e964be60620dc43))
- Use binary install mode for golangci-lint v2 ([25b47dc](https://github.com/aqasim81/database-migration-engine/commit/25b47dcc3388a42b8ae4abfc4e7928017e458484))
- Upgrade golangci-lint-action to v7 for golangci-lint v2 support ([9df7189](https://github.com/aqasim81/database-migration-engine/commit/9df718949ce34fab79eb403b1829d35adaf8485a))
- Install golangci-lint v2 from source in CI ([37788b5](https://github.com/aqasim81/database-migration-engine/commit/37788b5bd032f51ab63ef8df7307a3ed3aeaca96))
- Align CI golangci-lint version with local (v2.10.1) ([ac12734](https://github.com/aqasim81/database-migration-engine/commit/ac127341eba0fe6448398fc06cb387eb790b766b))
- Address phase 4 review feedback ([a21b994](https://github.com/aqasim81/database-migration-engine/commit/a21b994b9ec6e985e593eafa8522723cb74eddfd))
- Align pre-push coverage check with Makefile exclusion filter ([00d9a72](https://github.com/aqasim81/database-migration-engine/commit/00d9a7215a406f5ddd16ac81b8b2255d30919c4b))
- Address review feedback (phase 6) ([1a91dcc](https://github.com/aqasim81/database-migration-engine/commit/1a91dccc2b345e3e358e0ade5f03ec8872113845))

### Refactoring

- Rename containsConcurrentIndex to containsConcurrentOp ([af1689d](https://github.com/aqasim81/database-migration-engine/commit/af1689dc1d00e9a5bccc8a9f475d19183a2c8741))
- Unify SQL execution and rollback preamble in executor ([1b40740](https://github.com/aqasim81/database-migration-engine/commit/1b40740a20dd11c4ca4c50be4208094613ca8b47))
- Simplify planner code after review ([853e21f](https://github.com/aqasim81/database-migration-engine/commit/853e21f4400d1a044dafc88f9db3557d64bf5ce4))
- Simplify phase 8 code after review ([f2fe228](https://github.com/aqasim81/database-migration-engine/commit/f2fe22821f10362fc6f287ff9c946e558446fcc5))
- Extract shared CLI helpers and improve test cleanup ([1c2a50e](https://github.com/aqasim81/database-migration-engine/commit/1c2a50e16b7b5771feffc4f9e1542827c4bc2822))
- Extract setTestAppConfig helper to reduce test boilerplate ([7b7a634](https://github.com/aqasim81/database-migration-engine/commit/7b7a63495bb521559246220d0e60740f9b3fd6b5))
- Use setTestAppConfig helper in remaining apply tests ([831d30c](https://github.com/aqasim81/database-migration-engine/commit/831d30c908b108a5641552742cd3107fb8ed7eb9))

### Documentation

- Add MIT license ([cd224cf](https://github.com/aqasim81/database-migration-engine/commit/cd224cf2cac069b5aaf039aa6593052814a80989))
- Add README with architecture diagram and usage reference ([b987ca9](https://github.com/aqasim81/database-migration-engine/commit/b987ca915a653afc541e8c08e4aaddf31f7a4ed2))

### Tests

- Add CLI and analyzer tests to reach 91.9% coverage ([72c50dc](https://github.com/aqasim81/database-migration-engine/commit/72c50dc8685b56b7d44a0029a7c6ab865fbc3d9d))
- Add integration test infrastructure with testcontainers (phase 4) ([e08d7de](https://github.com/aqasim81/database-migration-engine/commit/e08d7de85c035210a77659d7cc35bdec709c5b3b))
- Add executor unit tests and apply lifecycle integration tests (phase 5) ([f401d4a](https://github.com/aqasim81/database-migration-engine/commit/f401d4a58e95815582c06ad30d8b60ae8fae59c6))
- Add rollback integration tests ([4314ab5](https://github.com/aqasim81/database-migration-engine/commit/4314ab5818983640b9880a665fa9980fc726f999))

### Chores

- Set up quality infrastructure with linting, coverage, and git hooks ([eb6aad2](https://github.com/aqasim81/database-migration-engine/commit/eb6aad27bb7899b434888513460e829f6042a567))
- Add uncommitted security hardening and config redaction ([56b6d27](https://github.com/aqasim81/database-migration-engine/commit/56b6d279e5cc31a131739e24a26630fe6b2cf101))
- Rename Go module path to github.com/aqasim81/database-migration-engine ([49949c3](https://github.com/aqasim81/database-migration-engine/commit/49949c3416628ebe594ca49276c43191661eca9e))
- Gitignore plans/ directory ([564ecb2](https://github.com/aqasim81/database-migration-engine/commit/564ecb2545deff7ab3f846167ae141a8906bd357))
- Add CI pipeline, update CLAUDE.md, fix .gitignore ([a70a683](https://github.com/aqasim81/database-migration-engine/commit/a70a683039b0a498ed4e710c1f6b653308ae42b7))
- Condense CLAUDE.md from 195 to 104 lines ([6976502](https://github.com/aqasim81/database-migration-engine/commit/69765025baf5af1c0dd32d22b71bf17fb6472fe0))
- Add pgx/v5 and testcontainers-go dependencies (phase 4) ([72a92e2](https://github.com/aqasim81/database-migration-engine/commit/72a92e2c459a28001084d9e8236f548a774f435c))
- Update coverage config and CI for database packages (phase 4) ([bd21cca](https://github.com/aqasim81/database-migration-engine/commit/bd21ccaddbcc5e6189cca2adc683f2eedb79e496))
- Update coverage config and CI for executor package (phase 5) ([ba70126](https://github.com/aqasim81/database-migration-engine/commit/ba70126ba1586244b299e787a8ee07d9a26d3693))
- Add goreleaser config and release workflow ([a01e380](https://github.com/aqasim81/database-migration-engine/commit/a01e380db2d45dafd49f448f0e91e58b92f0a80d))
- Consolidate CI into make audit plus integration job ([d2f5fdb](https://github.com/aqasim81/database-migration-engine/commit/d2f5fdb23da6ebe560f67f9e424ee1ca3bb8a7d9))
- Add pull request template ([e098368](https://github.com/aqasim81/database-migration-engine/commit/e0983685bae8349fa44714b359d0e445b62e1a96))


