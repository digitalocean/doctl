# Change Log

All changes to `doctl` will be documented in this file.

## [1.3.1] - 2016-07-13

### Changed
- #99 build test bin in out directory - @bryanl
- #104 Remove beta status for storage - @bryanl

### Fixed

- #100 password-reset, not power-reset - @aybabtme

## [1.3.0] - 2016-06-25

### Added
- #88 Add a --ssh-agent-forwarding-flag - @tbalthazar

### Changed
- #92 Rename drive to volume - @aybabtme
- #93 Extract token retrieval process - @bryanl
- #98 Remove output when deleting droplets - @bryanl

### Fixed
- #85 Don't report new release available when already installed - @andrewsomething
- #87 Update help output in documentation - @gmontalvoriv
- #97 User list images returns distributions - @bryanl

## [1.2.0] - 2016-06-07

### Added
- #37 Add a script to regenerate test mocks - @bryanl

### Changed
- #79 Ensure pagination is 1 indexed, and not 0 indexed - @jphines

### Fixed
- #68 Respect ssh-user flag - @vkurchatkin
- #69 Fix type in README - @aybabtme
- #70 Fix type in README - @aybabtme
- #74 Add more specific install instructions to README - @aybabtme
- #80 Fix a typo in usage text - @davidkuridza

## [1.1.0] - 2016-04-22

### Added
- #52 Add tagging commands - @bryanl
- #56 Add support for Drive beta - @aybabtme
- #58 Add support for beta features - @bryanl
- #63 Add ci build for windows - @bryanl

### Changed
- #53 Remove unused dependencies - @aybabtme
- #54 Rename root path - @aybabtme
- #57 Clean up version message - @aybabtme
- #61 Print drive columns if beta is enabled - @aybabtme

### Fixed
- #60 Disable tracing by default - @bryanl

## [1.0.2] - 2016-04-14

### Added
- #51 Adding change log - @bryanl

### Changed
- #41 All compute actions with `list` now have `ls` alias - @andrewsomething
- #44 Clean up references to doit - @aybabtme

### Fixed
- #49 Fix image argument to allow rebuilding droplets - @bryanl