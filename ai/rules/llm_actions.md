# AIC LLM Actions Protocol
This file teaches you how to execute actions on my system.

## Output Format
Your output must come line by line and be in a bash code block like so:

```bash
WE ARE NOT EXECUTING BASH
WE JUST PLACED THINGS IN A BASH CODE BLOCK
LINE BY LINE
```

This will allows us to easily copy/paste text from the LLM response.

## AIC: START;
If you want me to execute actions, you must start your entire response with `AIC: START;` on its own line. Only one `AIC: START;` is allowed per copiable series of text.


## AIC: END;
An `AIC: END;` is required at the end of an executale block. If an `AIC: END;` is not located, no commands will be executed.

## AIC: SHELL
An `AIC: SHELL` command allows us to execute shell commands on our system. For example, this would execute ls on our system:

```bash
AIC: START;
AIC: SHELL ls;
AIC; END;
```

## AIC: INSERT_START <path> <starting-line>;
Declaring `AIC: INSERT_START ./some_path.txt 2` will indicate you would like to insert some text in ./some_path.txt at line 2. This means line 2 will be shoved down (and the lines below it) to make things fit. For example, this would insert text starting at line 2 in `./some_path.txt`:

```bash
AIC: START;
AIC: INSERT_START ./some_file.txt 2;
I will be on line 2
I will be on line 3
AIC: INSERT_END;
AIC: END:
```

An `AIC: INSERT_END;` is required. If a matching one is not found an error will occur.

## AIC: REPLACE_START <path> <starting-line>; 
Declaring `AIC: REPLACE_START ./some_path.txt 2` will indicate you would like to replace some text in `./some_path.txt` starting from line 2. For example:

```bash
AIC: START;
AIC: REPLACE_START ./some_path.txt 2;
Hello, World!
This is replaced.
AIC: REPLACE_END;
AIC: END;
```

That would replace lines 2 and 3 in `./some_path.txt` with "Hello, World!\nThis is replaced."

An `AIC: REPLACE_END;` is required. If a matching one is not found an error will occur.


## AIC: DELETE_LINE <path> <starting-line> <ending-line>;
Declaring `AIC: DELETE_LINE ./some_path.txt 2 22;` will delete lines 2-22 (inclusive) in `./some_path.txt`