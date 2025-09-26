package editor

import (
	"strings"
	"testing"
	"veda/internal/config"
)

func TestIndentationDetection(t *testing.T) {
	t.Run("detect tabs", func(t *testing.T) {
		content := strings.Join([]string{
			"func main() {",
			"\tif true {",
			"\t\tfmt.Println(\"Hello\")",
			"\t}",
			"}",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentTabs {
			t.Errorf("Expected IndentTabs, got %v", info.Type)
		}
	})

	t.Run("detect 2 spaces", func(t *testing.T) {
		content := strings.Join([]string{
			"function test() {",
			"  if (true) {",
			"    console.log('hello');",
			"  }",
			"}",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", info.Type)
		}
		if info.Width != 2 {
			t.Errorf("Expected width 2, got %d", info.Width)
		}
	})

	t.Run("detect 4 spaces", func(t *testing.T) {
		content := strings.Join([]string{
			"def main():",
			"    if True:",
			"        print('Hello')",
			"    return 0",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", info.Type)
		}
		if info.Width != 4 {
			t.Errorf("Expected width 4, got %d", info.Width)
		}
	})

	t.Run("empty file defaults to spaces", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces for empty file, got %v", info.Type)
		}
		if info.Width != 4 {
			t.Errorf("Expected default width 4, got %d", info.Width)
		}
	})

	t.Run("C++ with 4 spaces", func(t *testing.T) {
		content := strings.Join([]string{
			"#include <iostream>",
			"",
			"class MyClass {",
			"public:",
			"    MyClass() {",
			"        std::cout << \"Constructor\" << std::endl;",
			"    }",
			"",
			"    void method() {",
			"        if (true) {",
			"            doSomething();",
			"        }",
			"    }",
			"private:",
			"    int value;",
			"};",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", info.Type)
		}
		if info.Width != 4 {
			t.Errorf("Expected width 4, got %d", info.Width)
		}
	})

	t.Run("config file with tabs", func(t *testing.T) {
		content := strings.Join([]string{
			"[user]",
			"\tname = Sasha",
			"\temail = sasha@example.com",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		te.CurrentBuffer().RealPath = "/tmp/home/.gitconfig"
		te.CurrentBuffer().FilePath = "/tmp/home/.gitconfig"

		info := te.detectIndentation()

		if info.Type != config.IndentTabs {
			t.Errorf("Expected IndentTabs for config file, got %v", info.Type)
		}
	})

	t.Run("C++ with 2 spaces", func(t *testing.T) {
		content := strings.Join([]string{
			"namespace example {",
			"  class Widget {",
			"  public:",
			"    Widget() : value_(0) {}",
			"    ",
			"    void process() {",
			"      for (int i = 0; i < 10; ++i) {",
			"        if (i % 2 == 0) {",
			"          handleEven(i);",
			"        }",
			"      }",
			"    }",
			"  };",
			"}",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", info.Type)
		}
		if info.Width != 2 {
			t.Errorf("Expected width 2, got %d", info.Width)
		}
	})

	t.Run("C with tabs", func(t *testing.T) {
		content := strings.Join([]string{
			"#include <stdio.h>",
			"",
			"int main(void) {",
			"\tprintf(\"Hello, World!\\n\");",
			"\t",
			"\tfor (int i = 0; i < 10; i++) {",
			"\t\tif (i % 2 == 0) {",
			"\t\t\tprintf(\"%d is even\\n\", i);",
			"\t\t}",
			"\t}",
			"\t",
			"\treturn 0;",
			"}",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentTabs {
			t.Errorf("Expected IndentTabs, got %v", info.Type)
		}
	})

	t.Run("Go with mixed content", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			"\t\"fmt\"",
			"\t\"strings\"",
			")",
			"",
			"func main() {",
			"\tdata := []string{",
			"\t\t\"apple\",",
			"\t\t\"banana\",",
			"\t\t\"cherry\",",
			"\t}",
			"\t",
			"\tfor _, item := range data {",
			"\t\tif strings.HasPrefix(item, \"a\") {",
			"\t\t\tfmt.Println(item)",
			"\t\t}",
			"\t}",
			"}",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentTabs {
			t.Errorf("Expected IndentTabs for Go code, got %v", info.Type)
		}
	})

	t.Run("mixed tabs and spaces picks majority", func(t *testing.T) {
		// Prefer the indentation style with majority support among reliable lines.
		content := strings.Join([]string{
			"void function1() {",
			"    doSomething();", // reliable (ends with ;) and 4 spaces
			"    doAnother();",   // reliable (ends with ;) and 4 spaces
			"}",
			"",
			"void function2() {",
			"\tdoWithTabs();", // reliable (ends with ;) and tab
			"}",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces || info.Width != 4 {
			t.Errorf("Expected IndentSpaces width 4 (majority), got %v width %d", info.Type, info.Width)
		}
	})

	t.Run("single space indents", func(t *testing.T) {
		// Some old or unusual code might use single spaces
		content := strings.Join([]string{
			"if true",
			" then",
			"  do_something",
			"   nested",
			" end",
			"end",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", info.Type)
		}
		// Should default to a reasonable width when single spaces are detected
		if info.Width != 2 && info.Width != 4 {
			t.Errorf("Expected width 2 or 4 for single space indents, got %d", info.Width)
		}
	})

	t.Run("inconsistent indentation", func(t *testing.T) {
		// Real-world files sometimes have inconsistent indentation
		content := strings.Join([]string{
			"function test() {",
			"  if (condition) {",
			"      // Someone used 6 spaces here",
			"      action1();",
			"    // And 4 spaces here",
			"    action2();",
			"  }",
			"}",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", info.Type)
		}
		// Should find GCD of 2, 4, 6 which is 2
		if info.Width != 2 {
			t.Errorf("Expected width 2 for inconsistent indents, got %d", info.Width)
		}
	})

	t.Run("C++ with access specifiers", func(t *testing.T) {
		// C++ often has access specifiers that might not be indented
		content := strings.Join([]string{
			"class Example {",
			"public:", // Often not indented
			"    Example() = default;",
			"    void publicMethod() {",
			"        doSomething();",
			"    }",
			"",
			"protected:", // Often not indented
			"    int protectedVar;",
			"",
			"private:", // Often not indented
			"    void privateMethod() {",
			"        if (condition) {",
			"            action();",
			"        }",
			"    }",
			"};",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", info.Type)
		}
		if info.Width != 4 {
			t.Errorf("Expected width 4, got %d", info.Width)
		}
	})

	t.Run("file with only unindented lines", func(t *testing.T) {
		// File with no indentation
		content := strings.Join([]string{
			"#ifndef HEADER_H",
			"#define HEADER_H",
			"",
			"// Forward declarations",
			"class MyClass;",
			"struct MyStruct;",
			"",
			"#endif // HEADER_H",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		// Should use default when no indentation is found
		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces for unindented file, got %v", info.Type)
		}
		if info.Width != 4 {
			t.Errorf("Expected default width 4, got %d", info.Width)
		}
	})

	t.Run("ast-parser.cpp pattern", func(t *testing.T) {
		// Actual content from ast-parser.cpp
		content := strings.Join([]string{
			"AST AstParser::parse_ast_files(const std::filesystem::path& ast_directory)",
			"{",
			"    AST ast;",
			"",
			"    if (!std::filesystem::exists(ast_directory)) {",
			"        std::cerr << \"AST directory does not exist: \" << ast_directory << std::endl;",
			"        return ast;",
			"    }",
			"",
			"    // Find all .ast.json files",
			"    for (const auto& entry : std::filesystem::directory_iterator(ast_directory)) {",
			"        if (entry.is_regular_file() && entry.path().extension() == \".json\" &&",
			"            entry.path().stem().extension() == \".ast\") {",
			"",
			"            logd << \"Parsing AST file: \" << entry.path();",
			"            if (!parse_ast_file(entry.path(), ast)) {",
			"                std::cerr << \"Failed to parse AST file: \" << entry.path() << std::endl;",
			"            }",
			"        }",
			"    }",
			"",
			"    return ast;",
			"}",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", info.Type)
		}
		if info.Width != 4 {
			t.Errorf("Expected width 4 for ast-parser.cpp pattern, got %d", info.Width)
		}
	})

	// New test: mixed 2- and 4-space indents should prefer 4 when 4 is majority
	t.Run("C mixed indent prefers 4 over singleton 2", func(t *testing.T) {
		content := strings.Join([]string{
			"#ifndef     _GNU_SOURCE",
			"  #define _GNU_SOURCE // for pthread_getaffinity_np()",
			"#endif  // !_GNU_SOURCE",
			"",
			"#include <stdio.h>",
			"",
			"int main()",
			"{",
			"    printf(\"Hello world\\n\");",
			"    return 0;",
			"}",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)
		info := te.detectIndentation()

		if info.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", info.Type)
		}
		if info.Width != 4 {
			t.Errorf("Expected width 4 for mixed 2/4 case, got %d", info.Width)
		}
	})
}

func TestIndentationCommands(t *testing.T) {
	t.Run("set noexpandtab", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "test content")

		// Start with default (4 spaces)
		if te.CurrentBuffer().Indentation.Type != config.IndentSpaces || te.CurrentBuffer().Indentation.Width != 4 {
			t.Fatalf("Expected default to be 4 spaces")
		}

		// Simulate entering command mode and typing :expandtab=false
		te.enterCommandMode()
		te.commandLine.SetText(":expandtab=false")
		te.executeCommand()

		if te.CurrentBuffer().Indentation.Type != config.IndentTabs {
			t.Errorf("Expected IndentTabs, got %v", te.CurrentBuffer().Indentation.Type)
		}
	})

	t.Run("set tabstop", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "test content")

		// Test :tabstop=2
		te.enterCommandMode()
		te.commandLine.SetText(":tabstop=2")
		te.executeCommand()

		if te.CurrentBuffer().Indentation.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces, got %v", te.CurrentBuffer().Indentation.Type)
		}
		if te.CurrentBuffer().Indentation.Width != 2 {
			t.Errorf("Expected width 2, got %d", te.CurrentBuffer().Indentation.Width)
		}
	})

	t.Run("set tabstop with equals", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "test content")

		// Test :tabstop=8
		te.enterCommandMode()
		te.commandLine.SetText(":tabstop=8")
		te.executeCommand()

		if te.CurrentBuffer().Indentation.Width != 8 {
			t.Errorf("Expected width 8, got %d", te.CurrentBuffer().Indentation.Width)
		}
	})

	t.Run("set expandtab", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "test content")

		// First set to tabs
		te.CurrentBuffer().SetIndentationType(config.IndentTabs)

		// Then use :expandtab=true
		te.enterCommandMode()
		te.commandLine.SetText(":expandtab=true")
		te.executeCommand()

		if te.CurrentBuffer().Indentation.Type != config.IndentSpaces {
			t.Errorf("Expected IndentSpaces after expandtab, got %v", te.CurrentBuffer().Indentation.Type)
		}
	})
}
