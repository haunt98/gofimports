# CHANGELOG

## v0.0.1 (2022-11-28)

### Added

- feat: implement format dir (2022-11-28)

- feat: write file actually (2022-11-28)

- feat: remove sort imports (2022-11-28)

- feat: ignore empty import (2022-11-27)

- feat: sort imports using default Go (wip) (2022-11-27)

- feat: print diff (2022-11-27)

- feat: actually print file from ast (2022-11-27)

- feat: rewrite all logic to single loop ast.Decl (wip) (2022-11-26)

- feat: support combine multi import decl (wip) (2022-11-26)

- feat: split local, company, third party imports (2022-11-26)

- feat: cache module name of path (2022-11-26)

- feat: get module name from path (2022-11-26)

- feat: query go.mod (2022-11-26)

- feat: sort import (wip) (2022-11-26)

- feat: parse imports and group imports std (2022-11-26)

- feat: simple format file (without actually format) (2022-11-25)

- feat: init Formatter (2022-11-25)

- feat: flags from gofmt (2022-11-24)

### Fixed

- fix: ignore not go file and go generated error (2022-11-28)

- fix: not copy import spec directly but use basic lit (2022-11-27)

- fix: force update ast decls when single import (2022-11-27)

- fix: formatter option missing value (2022-11-25)

### Others

- chore: add install, usage in README (2022-11-28)

- chore: add roadmap (2022-11-27)

- chore: print path when diff (2022-11-27)

- refactor: rewrite formatImportSpecs to eliminate dupe (2022-11-27)

- chore: remove useless check import empty (2022-11-27)

- refactor: pkgName -> moduleName (2022-11-26)

- chore: update README (2022-11-25)

- chore: add TODO (2022-11-25)

- refactor: remove regex code generated (2022-11-25)

- chore: update comment (2022-11-25)

- refactor: accept both write and diff (2022-11-25)

- chore: remove fmt.Println (2022-11-24)

- chore: fix lint (2022-11-24)

- chore: add MIT license (2022-11-24)

- chore: add github action, Makefile (2022-11-24)

- chore: init go.mod (2022-11-24)
