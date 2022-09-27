<p align="center">
  <h1 align="center"><b>promptkit</b></h1>
  <p align="center"><i>Interactive command line prompts with style!</i></p>
  <p align="center">
    <a href="https://github.com/erikgeiser/promptkit/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/erikgeiser/promptkit.svg?style=for-the-badge"></a>
    <a href="https://pkg.go.dev/github.com/erikgeiser/promptkit"><img alt="Go Doc" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge"></a>
    <a href="https://github.com/erikgeiser/promptkit/actions?workflow=Build"><img alt="GitHub Action: Build" src="https://img.shields.io/github/workflow/status/erikgeiser/promptkit/Build?label=Build&style=for-the-badge"></a>
    </br>
    <a href="https://github.com/erikgeiser/promptkit/actions?workflow=Check"><img alt="GitHub Action: Check" src="https://img.shields.io/github/workflow/status/erikgeiser/promptkit/Check?label=Check&style=for-the-badge"></a>
    <a href="/LICENSE.md"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge"></a>
    <a href="https://goreportcard.com/report/github.com/erikgeiser/promptkit"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/erikgeiser/promptkit?style=for-the-badge"></a>
  </p>
</p>

Promptkit is a collection of common command line prompts for interactive
programs. Each prompts comes with sensible defaults, re-mappable key bindings
and many opportunities for heavy customization.

---

**Disclaimers:**
* The API of library is not yet stable. Expect significant changes in minor
  versions before `v1.0.0`.
* Windows is currently not explicitly supported because input events are dropped
  when running multiple prompts sequentially due to a bug in
  [bubbletea](https://github.com/charmbracelet/bubbletea). See
  https://github.com/charmbracelet/bubbletea/pull/140 and
  https://github.com/charmbracelet/bubbletea/issues/121 for more information.

---

## Selection

Selection with filter and pagination support: [Example Code](https://github.com/erikgeiser/promptkit/blob/main/examples/selection/main.go)

<a href="https://asciinema.org/a/8co2qSgAIxRZBJzvX5iZXwUqF" target="_blank"><img src="https://asciinema.org/a/8co2qSgAIxRZBJzvX5iZXwUqF.svg" /></a>

---

The selection prompt is highly customizable and also works well with custom
types which for example enables filtering only on custom fields: [Example Code](https://github.com/erikgeiser/promptkit/blob/main/examples/selection_custom/main.go)

<a href="https://asciinema.org/a/Uw7QjXK7nZ0eHmigqIzdDXk3C" target="_blank"><img src="https://asciinema.org/a/Uw7QjXK7nZ0eHmigqIzdDXk3C.svg" /></a>

---

## Text Input

A text input that supports editable default values: [Example Code](https://github.com/erikgeiser/promptkit/blob/main/examples/textinput/main.go)

<a href="https://asciinema.org/a/xUudX97RAXNnHMkArASH4Ccgv" target="_blank"><img src="https://asciinema.org/a/xUudX97RAXNnHMkArASH4Ccgv.svg" /></a>

---

Custom validation is also supported: [Example Code](https://github.com/erikgeiser/promptkit/blob/main/examples/textinput_custom/main.go)

<a href="https://asciinema.org/a/FF14DYA8WtEtRjdPkcllAJk9p" target="_blank"><img src="https://asciinema.org/a/FF14DYA8WtEtRjdPkcllAJk9p.svg" /></a>

---

The text input can also be used in hidden mode for password prompts: [Example Code](https://github.com/erikgeiser/promptkit/blob/main/examples/textinput_hidden/main.go)

<a href="https://asciinema.org/a/k2KTLG49OWWQ3AofrGSzWVmkL" target="_blank"><img src="https://asciinema.org/a/k2KTLG49OWWQ3AofrGSzWVmkL.svg" /></a>

---

## Confirmation Prompt

A confirmation prompt for binary questions: [Example Code](https://github.com/erikgeiser/promptkit/blob/main/examples/confirmation/main.go)

<a href="https://asciinema.org/a/dpQHPP22ceylJGbSthAekZwBB" target="_blank"><img src="https://asciinema.org/a/dpQHPP22ceylJGbSthAekZwBB.svg" /></a>

## Widget

The prompts in this library can also be used as [bubbletea](https://github.com/charmbracelet/bubbletea) widgets: [Example Code](https://github.com/erikgeiser/promptkit/blob/main/examples/bubbletea_widget/main.go)

## Acknowledgements

This library is built on top of many great libraries, especially the following:

- https://github.com/charmbracelet/bubbletea
- https://github.com/charmbracelet/bubbles
- https://github.com/muesli/termenv
- https://github.com/muesli/reflow
