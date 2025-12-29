# AIC LLM Actions Protocol (Clipboard Agent Actions)

This file teaches an LLM how to produce **machine-interpretable “action blocks”** for the `aic` tool running on the user's computer.

When the user copies text to their clipboard, `aic` watches the clipboard and will **only** execute actions if the clipboard content is a valid AIC action block.

---

## 0) Most Important Rule: Every “Action Response” Must Be One Big Action Block

When you (the LLM) intend to make *any* change on the user's machine (edit files, run shell commands, etc.), your **entire response** must be exactly one AIC action block:

- The response must start with `AIC: START;` (as the first line of the response)
- The response must end with `AIC: END;` (as the last line of the response)
- Everything between those lines must be **only** AIC directives / content lines for directives
- **No** prose, explanations, markdown, code fences, headings, bullets, or extra characters outside the block

If you include *anything* outside the block (even a single character), the user may copy it and cause:
- the block to be rejected, or
- the user to copy an unsafe / unintended payload.

### Action response vs. normal response
- **Action response:** one big `AIC: START; ... AIC: END;` block only.
- **Normal response:** any explanation or discussion with **no** AIC block.

If the user asks for both explanation *and* actions, prefer:
1) provide a **normal response** explaining what you would do, *or*
2) provide an **action response** only (and keep it minimal/safe).

Do **not** mix explanation text with an action block in the same message.

---

## 1) The One Rule That Matters: Exact START/END Envelope

For `aic` to process your response, the clipboard text **must**:

- **Start with** the exact first line: `AIC: START;`
- **End with** the exact last line: `AIC: END;`
- Contain **no extra characters** (including code fences) before `AIC: START;` or after `AIC: END;`

✅ Valid (this is the *entire* clipboard content):
AIC: START;
AIC: SHELL ls;
AIC: END;

❌ Invalid (has a code fence, so the first line isn’t `AIC: START;`):
```bash
AIC: START;
AIC: SHELL ls;
AIC: END;
```

❌ Invalid (END typo):
AIC: END:

❌ Invalid (START missing semicolon):
AIC: START

### Whitespace rules (be strict)
- Put `AIC: START;` on its own line with **no leading/trailing spaces**.
- Put `AIC: END;` on its own line with **no leading/trailing spaces**.
- Do not add blank lines before START or after END.
- Inside the block, blank lines are allowed, but avoid them unless needed.

---

## 2) Command Line Grammar

Inside the envelope, lines are processed top-to-bottom.

### General command format
Most commands are a single line:

AIC: <COMMAND> <ARGS...>;

Where:
- The line begins with `AIC:` (case-insensitive for most commands; see Start/End above).
- Commands **should end with a semicolon** `;` to be unambiguous and consistent.
- Arguments are separated by spaces.
- Paths are **relative to the project root** (the directory that contains `./ai/`).

---

## 3) Available Actions

### 3.1) AIC: SHELL — run a shell command

**Purpose:** Execute a shell command via `sh -c`.

**Syntax:**
AIC: SHELL <shell command>;

**Examples:**
AIC: START;
AIC: SHELL ls;
AIC: END;

AIC: START;
AIC: SHELL ls; pwd; echo "done";
AIC: END;

Notes:
- The `<shell command>` may itself contain multiple commands separated by `;`.
- Include the **final** trailing `;` so the line is clearly terminated.
- Do not include backticks or triple backticks in the action block.

---

### 3.2) AIC: REPLACE_START / REPLACE_END — overwrite file lines

**Purpose:** Replace lines in a file starting at a given 1-based line number.

**Syntax:**
AIC: REPLACE_START <relative_path> <start_line>;
<replacement line 1>
<replacement line 2>
...
AIC: REPLACE_END;

Rules:
- `<start_line>` is **1-based** (first line is 1).
- All lines between `REPLACE_START` and `REPLACE_END` become the replacement content.
- The replacement overwrites exactly as many lines as you provide.
- If `REPLACE_END;` appears without a preceding `REPLACE_START`, `aic` will error.
- If a `REPLACE_START` is opened and no matching `REPLACE_END;` is found, `aic` will error.

**Example:**
AIC: START;
AIC: REPLACE_START ./some_file.py 22;
print("new line 22")
print("new line 23")
AIC: REPLACE_END;
AIC: END;

---

### 3.3) AIC: INSERT_START / INSERT_END — insert file lines

**Purpose:** Insert lines into a file at a given 1-based line number.

**Syntax:**
AIC: INSERT_START <relative_path> <line_num>;
<inserted line 1>
<inserted line 2>
...
AIC: INSERT_END;

Rules:
- `<line_num>` is **1-based**.
- Insert at line N means the inserted text becomes the new lines starting at N.
- Same block pairing rules as REPLACE apply (orphaned END = error).

**Example:**
AIC: START;
AIC: INSERT_START ./README.md 5;
## New Section
Here is some new content.
AIC: INSERT_END;
AIC: END;

---

### 3.4) AIC: DELETE_LINE — delete a range of lines

**Purpose:** Delete a line range (inclusive) from a file.

**Syntax:**
AIC: DELETE_LINE <relative_path> <start_line> <end_line>;

Rules:
- `<start_line>` and `<end_line>` are **1-based**, inclusive.
- If the range is invalid or out-of-bounds, `aic` will error.

**Example:**
AIC: START;
AIC: DELETE_LINE ./some_file.py 10 20;
AIC: END;

---

## 4) How to Respond (LLM Behavior Rules)

When the user asks you to trigger actions:

1. Decide whether an action is actually needed.
2. If yes, output **only** the action block:
   - First line: `AIC: START;`
   - Last line: `AIC: END;`
   - No extra commentary outside the block.
3. Prefer **minimal, explicit** actions.
4. Use **project-relative paths** (e.g., `./pkg/foo.go`, `./ai/rules/llm_actions.md`).

If the user requests explanations *and* actions, provide the explanation **outside** the action block in a separate message, or ask them whether they want a “safe no-actions” explanation. (If you include any text outside the action block, it will not be executed — but it may prevent the block from being detected if it appears before START or after END.)

---

## 5) Security & Safety (IMPORTANT)

`AIC: SHELL` can run arbitrary commands on the user's machine. Treat it as **high-risk**.

### Safe defaults
- Prefer file edits via `REPLACE_START/INSERT_START/DELETE_LINE` over `SHELL` whenever possible.
- If you must use `SHELL`, keep commands **short, specific, and reversible**.
- Avoid destructive commands unless the user explicitly requests them and you are confident they understand the impact.

### Never do this
Do **not** generate commands that:
- Exfiltrate secrets (e.g., reading SSH keys, browser cookies, password stores).
- Transmit files or environment variables to the network (curl/wget/scp/nc).
- Modify system security settings, install persistence, or create backdoors.
- Delete large directories or wipe disks (`rm -rf`, `diskutil eraseDisk`, etc.) unless the user explicitly asked and confirmed.

### Recommended “safe pattern” for shell usage
- Use read-only commands by default (`ls`, `pwd`, `cat`, `grep`, `git status`).
- Print what you’re doing (`echo ...`) if it helps the user understand.
- Avoid chaining many commands unless needed.

---

## 6) Troubleshooting Checklist

If actions are not triggering, check:

- Is the **first line exactly** `AIC: START;` ?
- Is the **last line exactly** `AIC: END;` ?
- Did you accidentally include a code fence (```), markdown, or extra text?
- Do command lines end with `;` ?
- Are file paths relative to the repo root (the directory containing `./ai/`)?

---

## 7) Example: “Run ls”

AIC: START;
AIC: SHELL ls;
AIC: END;
