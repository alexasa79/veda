I would like to improve the algorithm we use for:
1. Figuring out which argument of a function we're currently looking at as we
   browse through the code.
2. Figuring out when to automatically close the function's signature.

The basic algorithm I want to try is the following. We request a signature from
the LSP. We can be anywhere in "f1(arg1, arg2, ...)". Let's say we sent a
request to get the signature and we received it. We don't know where in the
function call we are. The cursor could be pointing the at the name of the
function or at any one of the arguments. Moreover, the number of arguments can
be wrong. From LSP we received the name of the current function, so we know that
we must have "f1" somewhere near. We will start by searching forward until we
find something that is not alphanumeric character - either space or punctuation,
EOL or EOF. Then we will search backwards until we find "f1" (the function
name). This is going to be our anchor. Once we found the name of the function,
we know the position of the cursor relative to the argument list. Next, we will
search for open paren. Once found, we will look for matching closing paren. Once
found, we got our argument list. If we are in the argument list and the current
argument matches the list of arguments (e.g., if the function has 10 arguments
and we're in argument before 10), then we will show the signature, highlighting
the current argument. Otherwise, we will fallback to just showing the signature.

Start with implementing unit tests that check various conditions. For example,
we're pointing at "f1" and we want to identify where the function call starts
and that we're outside of the argument list. If the cursor as on arg2, we want
to identify that we're on the second argument, etc. I want you to pay special
attention to cases where the function being called receives more arguments than
we pass, receives less arguments than we pass (i.e., we give it more arguments
than needed), and a case where we have function calls and parenthesized
expressions inside of the argument list.
