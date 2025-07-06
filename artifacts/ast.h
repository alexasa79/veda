#pragma once

#include <string>
#include <vector>
#include <set>
#include <map>
#include <memory>
#include <optional>
#include <iostream>
#include "json.hpp"
#include "expression.h"
#include "z3-solver.h"
#include "symbolic-variable.h"

using json = nlohmann::json;

// Forward declarations
class Function;
class LinearEquation;

// Represents a scope that contains variables (function arguments and local variables)
class Scope
{
private:
    // Map from variable name to Variable pointer
    // Using shared_ptr to ensure Variables remain valid throughout the scope's lifetime
    std::map<std::string, std::shared_ptr<Variable>> variables;

    // Function this scope belongs to
    const Function* function;

public:
    Scope(const Function* func = nullptr) : function(func) {}

    // Add a variable to the scope
    void add_variable(std::shared_ptr<Variable> var);

    // Look up a variable by name
    std::shared_ptr<Variable> lookup(const std::string& name) const;

    // Check if a variable exists in the scope
    bool contains(const std::string& name) const;

    // Get all variables in the scope
    const std::map<std::string, std::shared_ptr<Variable>>& get_variables() const {
        return variables;
    }

    // Get the function this scope belongs to
    const Function* get_function() const { return function; }

    // Set the function this scope belongs to
    void set_function(const Function* func) { function = func; }
};

// Represents a variable (function argument or local variable)
// This is the static version - it doesn't have a value
class Variable
{
public:
    enum class Kind {
        FunctionArgument,
        LocalVariable
    };

private:
    Kind kind;
    std::string name;
    std::string type;  // C type as string
    const Function* function;  // Function this variable belongs to
    int argument_index;  // For function arguments, which parameter (0-based)
    std::string filename;  // Source file name
    int line_number;  // Line number where variable is declared/used
    const json* init_json_node = nullptr;  // JSON node for initialization expression (if any)

public:
    Variable(const std::string& name, const std::string& type,
             const Function* func, int arg_index,
             const std::string& filename, int line_number)
        : kind(Kind::FunctionArgument), name(name), type(type),
          function(func), argument_index(arg_index),
          filename(filename), line_number(line_number) {}

    // Constructor for local variables
    Variable(const std::string& name, const std::string& type,
             const Function* func, const std::string& filename, int line_number,
             const json* init_node = nullptr)
        : kind(Kind::LocalVariable), name(name), type(type),
          function(func), argument_index(-1),
          filename(filename), line_number(line_number), init_json_node(init_node) {}

    Kind get_kind() const { return kind; }
    const std::string& get_name() const { return name; }
    const std::string& get_type() const { return type; }
    const Function* get_function() const { return function; }
    int get_argument_index() const { return argument_index; }
    const std::string& get_filename() const { return filename; }
    int get_line_number() const { return line_number; }
    const json* get_init_json_node() const { return init_json_node; }

    std::string to_string() const;
};

struct SourceLocation
{
    std::string file;
    int line = 0;
    int column = 0;

    bool operator<(const SourceLocation& other) const
    {
        if (file != other.file) return file < other.file;
        if (line != other.line) return line < other.line;
        return column < other.column;
    }

    bool operator==(const SourceLocation& other) const
    {
        return file == other.file && line == other.line && column == other.column;
    }
};

struct PathStep
{
    SourceLocation location;           // Where the call happens
    const Function* caller;            // nullptr for entry points
    const Function* called;            // Function being called
    const json* call_json_node = nullptr; // JSON node for the call expression

    PathStep(const SourceLocation& loc, const Function* caller_func, const Function* called_func, const json* call_node)
        : location(loc), caller(caller_func), called(called_func), call_json_node(call_node) {}
};

class CodePath {
private:
    // Map to track created symbolic variables for this path
    std::map<std::string, std::shared_ptr<SymbolicVariable>> symbolic_vars;

public:
    std::vector<PathStep> steps;  // Single vector instead of parallel arrays
    std::vector<std::shared_ptr<LinearEquation>> constraints;  // Track symbolic constraints

    void add_step(const PathStep& step);
    void add_constraint(std::shared_ptr<LinearEquation> constraint);
    size_t length() const { return steps.size(); }
    bool empty() const { return steps.empty(); }
    const std::vector<std::shared_ptr<LinearEquation>>& get_constraints() const { return constraints; }
    const PathStep& get_step(int index) const;

    // Get or create a symbolic variable for this path
    std::shared_ptr<SymbolicVariable> get_or_create_symbolic_variable(const Variable* var,
                                                                     const std::string& symbolic_name);
};

class Function
{
public:
    std::string name;
    SourceLocation definition_location;
    std::vector<std::shared_ptr<Variable>> arguments;
    std::string return_type;
    bool defined = true;  // true for function definitions, false for external declarations
    const json* json_node = nullptr;  // Pointer to the JSON node representing this function
    Scope scope;  // Scope containing all variables in this function

    Function(const std::string& function_name);
    Function(const std::string& function_name, const SourceLocation& location);

    bool operator<(const Function& other) const;
    bool operator==(const Function& other) const;
};

class FunctionCall
{
public:
    std::string caller_function;
    std::string called_function;
    SourceLocation call_location;
    const json* json_node = nullptr;  // Pointer to the JSON node representing this function call

    FunctionCall(const std::string& caller, const std::string& called, const SourceLocation& location);

    bool operator<(const FunctionCall& other) const;
    bool operator==(const FunctionCall& other) const;
};

class AST
{
private:
    std::set<Function> functions;
    std::vector<FunctionCall> function_calls;
    std::vector<std::shared_ptr<json>> json_documents;  // Store JSON documents to keep pointers valid

public:
    // Add functions
    void add_function(const Function& function);
    void add_function_call(const FunctionCall& call);
    void add_json_document(std::shared_ptr<json> doc) { json_documents.push_back(doc); }

    // Getters
    const std::set<Function>& get_functions() const { return functions; }
    const std::vector<FunctionCall>& get_function_calls() const { return function_calls; }

    // Get defined functions only
    std::set<Function> get_defined_functions() const;

    // Get external functions only
    std::set<Function> get_external_functions() const;

    // Check if a function is defined in this AST
    bool is_function_defined(const std::string& name) const;

    // Call graph functionality
    std::vector<std::string> get_called_functions(const std::string& function_name) const;
    std::vector<std::string> get_calling_functions(const std::string& function_name) const;
    std::vector<CodePath> find_call_paths_to(const std::string& target_function) const;
    std::vector<CodePath> find_call_paths_with_constraint(const std::string& target_function,
                                                         const std::string& arg_name,
                                                         long value) const;

    // Check if constraints along a path are satisfiable using z3
    bool check_path_satisfiability(const CodePath& path) const;

    // Generate SMT-LIB string for constraints along a path
    std::string generate_smtlib_for_path(const CodePath& path) const;
    void print_call_graph() const;

    // Helper method to extract argument value from CallExpr JSON node
    std::pair<bool, long> extract_argument_value(const json* call_node, int arg_index) const;

    // Helper method to extract argument expression from CallExpr JSON node
    std::shared_ptr<Expression> extract_argument_expression(const json* call_node, int arg_index) const;

    // Parse an expression from AST JSON node
    std::shared_ptr<Expression> parse_expression(const json& node) const;

    // Evaluate a constant expression to get its value
    std::optional<long> evaluate_expression(const std::shared_ptr<Expression>& expr) const;

private:
    // Helper method to build Model for a code path with constraint
    Model build_model_for_path(CodePath& path,
                              const std::string& target_function,
                              const std::string& arg_name,
                              long value) const;

    // Helper to find a Variable by name in a function
    const Variable* find_variable(const std::string& function_name, const std::string& var_name) const;

    // Helper to process variable constraints recursively
    void process_variable_constraints(Model& model,
                                    CodePath& path,
                                    int location_index,
                                    const std::string& var_name,
                                    const std::string& function_name) const;

    // Helper to find all variables in an expression
    std::set<std::string> find_variables_in_expression(std::shared_ptr<Expression> expr) const;

    // Helper to convert expression with variable prefix
    std::shared_ptr<Expression> convert_expression_with_prefix(std::shared_ptr<Expression> expr,
                                                              const std::string& var_prefix,
                                                              CodePath& path) const;
};

// Stream operators for printing
std::ostream& operator<<(std::ostream& os, const CodePath& path);
std::ostream& operator<<(std::ostream& os, const PathStep& step);
std::ostream& operator<<(std::ostream& os, const SourceLocation& location);
