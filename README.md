# AIC

A local Context Bridge for Large Language Model prompting. AIC watches
your files, compiles context, and prepares prompts automatically.

## Overview

AIC is a background tool designed to streamline the workflow of copying
code and context into AI chat interfaces. Instead of manually copying
files, running shell commands, and pasting them into a browser, AIC
automates the process.

It watches a specific file (`ai/prompt.md`). When you save that file,
AIC:

1.  Executes defined shell commands (optional).
2.  Reads your current system clipboard.
3.  Collects global project rules.
4.  Combines everything into a structured prompt.
5.  **Writes the final result back to your clipboard**, ready to paste.

## Project Structure

    ./ai
    ├── prompt.md
    ├── rules/
    └── prompts/

## Usage

Edit `ai/prompt.md` using the format below:

    ---
    ls -la ./pkg
    tree ./src
    ---
    Refactor the file listed in the clipboard based on the tree structure above.

## Key Features

-   **Auto-Stashing** -- every prompt is saved.
-   **History Pruning** -- keeps the last 100 prompts.
-   **GitIgnore Aware** -- respects .gitignore.
-   **Debounced Watcher** -- prevents double triggers.
