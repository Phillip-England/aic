# LLM MODEL THIS IS MY PROMPT:
Okay I would like to be able to run shell commands, get the std out, and then include it in the prompt output. This will be done using dollar sign tokens. I can say $SHELL(ls) and that should return the output of the ls command in my actual prompt output.

This enables me to include binary output in a prompt which is very useful.

Notive how the shell command does not require quotes, just anything between the parenthesis.

this will require us to change how dollar sign tokesn are parsed. They start at a dollar sign, but the parser must be smart enough to include the parenthesis if they exist and if parenthesis do exist the parser should be smart enough to not mess up on the way parenthesis are parsed. We might need to test this to ensure parenthesis in weird ways cannot be executed.

Okay, this is alot, but I know you can do it. This will allow us to run shell commands from a file and collect output.

@.