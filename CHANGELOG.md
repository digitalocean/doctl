# Change Log

All changes to `doctl` will be documented in this file.

## [1.2.0] - 2016-06-07

### Added
- #37 Add a script to regenerate test mocks

### Changed
- #79 Ensure pagination is 1 indexed, and not 0 indexed

### Fixed
- #68 Respect ssh-user flag
- #69 Fix type in README
- #70 Fix type in README
- #74 Add more specific install instructions to README
- #80 Fix a typo in usage text

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