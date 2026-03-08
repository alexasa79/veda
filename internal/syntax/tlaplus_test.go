package syntax

import "testing"

func TestTLAPlusSyntaxHighlighting(t *testing.T) {
	h := NewHighlighter()

	if err := h.SetLanguage("tlaplus"); err != nil {
		t.Fatalf("failed to set language to tlaplus: %v", err)
	}

	tlaCode := `---- MODULE Transfer ----
EXTENDS Naturals, Sequences

VARIABLES balance, history

Init ==
    /\ balance = [a |-> 100, b |-> 100]
    /\ history = <<>>

Transfer(from, to, amount) ==
    /\ amount > 0
    /\ balance[from] >= amount
    /\ balance' = [balance EXCEPT ![from] = @ - amount, ![to] = @ + amount]
    /\ history' = Append(history, amount)

\* This is a line comment
(* This is
   a block comment *)

TypeInvariant ==
    /\ \A x \in {"a", "b"} : balance[x] >= 0
    /\ balance["a"] + balance["b"] = 200

THEOREM Init => TypeInvariant

(* --algorithm TransferAlgo
variables bal = [a |-> 100, b |-> 100];
begin
    Transfer:
        if bal["a"] >= 50 then
            bal["a"] := bal["a"] - 50;
            bal["b"] := bal["b"] + 50;
        end if;
end algorithm; *)
====
`

	h.ParseFull([]byte(tlaCode))

	type expectation struct {
		text      string
		tokenType TokenType
	}

	tokens := h.GetTokensForString(tlaCode)

	expectations := []expectation{
		// TLA+ keywords
		{"MODULE", TokenTypeKeyword},
		{"EXTENDS", TokenTypeKeyword},
		{"VARIABLES", TokenTypeKeyword},
		{"EXCEPT", TokenTypeKeyword},
		{"THEOREM", TokenTypeKeyword},

		// Operator definitions -> function
		{"Init", TokenTypeFunction},
		{"Transfer", TokenTypeFunction},
		{"TypeInvariant", TokenTypeFunction},

		// Variables
		{"balance", TokenTypeVariable},
		{"history", TokenTypeVariable},

		// Strings
		{"\"a\"", TokenTypeString},

		// Numbers
		{"100", TokenTypeNumber},
		{"0", TokenTypeNumber},
		{"200", TokenTypeNumber},

		// Comments
		{"\\* This is a line comment", TokenTypeComment},

		// PlusCal keywords
		{"algorithm", TokenTypeKeyword},
		{"variables", TokenTypeKeyword},
		{"begin", TokenTypeKeyword},
		{"if", TokenTypeKeyword},
		{"then", TokenTypeKeyword},
		{"end", TokenTypeKeyword},
	}

	for _, exp := range expectations {
		found := false
		for _, token := range tokens {
			if token.Start < 0 || token.End > len(tlaCode) || token.Start >= token.End {
				continue
			}
			tokenText := tlaCode[token.Start:token.End]
			if tokenText == exp.text {
				if token.TokenType != exp.tokenType {
					t.Errorf("wrong token type for %q: got %v, want %v", exp.text, token.TokenType, exp.tokenType)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("token %q not found", exp.text)
		}
	}
}
