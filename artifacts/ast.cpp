#include "ast.h"
#include "log.h"
#include "symbolic-variable.h"
#include "z3-solver.h"
#include <iostream>
#include <map>
#include <set>
#include <functional>
#include <sstream>
#include <cstdlib>
#include <cstdio>
#include <unistd.h>

// Scope class implementation
void Scope::add_variable(std::shared_ptr<Variable> var)
{
    if (var) {
        variables[var->get_name()] = var;
    }
}

std::shared_ptr<Variable> Scope::lookup(const std::string& name) const
{
    auto it = variables.find(name);
    if (it != variables.end()) {
        return it->second;
    }
    return nullptr;
}

bool Scope::contains(const std::string& name) const
{
    return variables.find(name) != variables.end();
}

// Variable class implementation
std::string Variable::to_string() const
{
    std::ostringstream oss;
    oss << "Variable{name=\"" << name << "\", type=\"" << type << "\"";
    if (kind == Kind::FunctionArgument) {
        oss << ", arg_index=" << argument_index;
        if (function) {
            oss << ", function=\"" << function->name << "\"";
        }
    }
    oss << ", location=" << filename << ":" << line_number << "}";
    return oss.str();
}

Function::Function(const std::string& function_name)
    : name(function_name), defined(true), scope(this)
{
}

Function::Function(const std::string& function_name, const SourceLocation& location)
    : name(function_name), definition_location(location), defined(true), scope(this)
{
}

bool Function::operator<(const Function& other) const
{
    return name < other.name;
}

bool Function::operator==(const Function& other) const
{
    return name == other.name;
}

FunctionCall::FunctionCall(const std::string& caller, const std::string& called, const SourceLocation& location)
    : caller_function(caller), called_function(called), call_location(location)
{
}

bool FunctionCall::operator<(const FunctionCall& other) const
{
    if (caller_function != other.caller_function) {
        return caller_function < other.caller_function;
    }
    return called_function < other.called_function;
}

bool FunctionCall::operator==(const FunctionCall& other) const
{
    return caller_function == other.caller_function && called_function == other.called_function;
}

void AST::add_function(const Function& function)
{
    functions.insert(function);
}

void AST::add_function_call(const FunctionCall& call)
{
    function_calls.push_back(call);
}

std::set<Function> AST::get_defined_functions() const
{
    std::set<Function> defined;
    for (const auto& func : functions) {
        if (func.defined) {
            defined.insert(func);
        }
    }
    return defined;
}

std::set<Function> AST::get_external_functions() const
{
    std::set<Function> external;
    for (const auto& func : functions) {
        if (!func.defined) {
            external.insert(func);
        }
    }
    return external;
}

bool AST::is_function_defined(const std::string& name) const
{
    for (const auto& func : functions) {
        if (func.name == name && func.defined) {
            return true;
        }
    }
    return false;
}

void CodePath::add_step(const PathStep& step)
{
    steps.push_back(step);
}

void CodePath::add_constraint(std::shared_ptr<LinearEquation> constraint)
{
    constraints.push_back(constraint);
}

const PathStep& CodePath::get_step(int index) const
{
    if (index >= 0 && index < static_cast<int>(steps.size())) {
        return steps[index];
    }
    // Return a static empty step for out-of-bounds access
    static PathStep empty_step(SourceLocation{}, nullptr, nullptr, nullptr);
    return empty_step;
}

std::shared_ptr<SymbolicVariable> CodePath::get_or_create_symbolic_variable(const Variable* var,
                                                                            const std::string& symbolic_name)
{
    auto it = symbolic_vars.find(symbolic_name);
    if (it != symbolic_vars.end()) {
        return it->second;
    }

    // Create a new symbolic variable
    auto sym_var = std::make_shared<SymbolicVariable>(var, symbolic_name);
    symbolic_vars[symbolic_name] = sym_var;
    return sym_var;
}

std::vector<std::string> AST::get_called_functions(const std::string& function_name) const
{
    std::vector<std::string> called;
    for (const auto& call : function_calls) {
        if (call.caller_function == function_name) {
            called.push_back(call.called_function);
        }
    }
    return called;
}

std::vector<std::string> AST::get_calling_functions(const std::string& function_name) const
{
    std::vector<std::string> callers;
    for (const auto& call : function_calls) {
        if (call.called_function == function_name) {
            callers.push_back(call.caller_function);
        }
    }
    return callers;
}

void AST::print_call_graph() const
{
    std::cout << "=== Call Graph ===" << std::endl;
    std::cout << "Functions (" << functions.size() << "):" << std::endl;

    std::cout << "\nFunction Calls (" << function_calls.size() << "):" << std::endl;

    // Build call sequence by finding chains
    std::map<std::string, std::string> call_map;
    std::set<std::string> all_functions;

    // Create a map from function name to line number
    std::map<std::string, int> function_lines;
    for (const auto& function : functions) {
        function_lines[function.name] = function.definition_location.line;
    }

    for (const auto& call : function_calls) {
        call_map[call.caller_function] = call.called_function;
        all_functions.insert(call.caller_function);
        all_functions.insert(call.called_function);
    }

    // Find entry points (functions that are not called by others)
    std::set<std::string> entry_points;
    for (const auto& func : all_functions) {
        bool is_called = false;
        for (const auto& call : function_calls) {
            if (call.called_function == func) {
                is_called = true;
                break;
            }
        }
        if (!is_called) {
            entry_points.insert(func);
        }
    }

    // Helper function to format function name with line number
    auto format_function_name = [&function_lines](const std::string& func_name) -> std::string {
        auto it = function_lines.find(func_name);
        if (it != function_lines.end() && it->second > 0) {
            return func_name + ":" + std::to_string(it->second);
        }
        return func_name;
    };

    // Print call sequences starting from entry points
    for (const auto& entry : entry_points) {
        std::string current = entry;
        std::cout << "  " << format_function_name(current);

        while (call_map.find(current) != call_map.end()) {
            current = call_map[current];
            std::cout << " -> " << format_function_name(current);
        }
        std::cout << std::endl;
    }
}

std::vector<CodePath> AST::find_call_paths_to(const std::string& target_function) const
{
    std::vector<CodePath> all_paths;

    // Find the target function
    const Function* target_func = nullptr;
    for (const auto& func : functions) {
        if (func.name == target_function) {
            target_func = &func;
            break;
        }
    }

    if (!target_func) {
        return all_paths; // Target function not found
    }

    // Build a map from function name to Function pointer
    std::map<std::string, const Function*> function_map;
    for (const auto& func : functions) {
        function_map[func.name] = &func;
    }

    // Helper lambda for DFS - now uses PathStep
    std::function<void(const std::string&, CodePath&, std::set<std::string>&)> dfs =
        [&](const std::string& current, CodePath& current_path, std::set<std::string>& visited) {

        if (current == target_function) {
            // Found a path to the target
            all_paths.push_back(current_path);
            return;
        }

        // Check if already visited to avoid cycles
        if (visited.find(current) != visited.end()) {
            return;
        }

        visited.insert(current);

        // Get all function calls made by the current function
        for (const auto& call : function_calls) {
            if (call.caller_function == current) {
                // Find the caller and called functions
                const Function* caller_func = function_map[call.caller_function];
                const Function* called_func = function_map[call.called_function];

                // Add the call step to the path
                logt << "Creating PathStep with json_node: " << call.json_node
                     << " for call from " << call.caller_function << " to " << call.called_function;
                PathStep step(call.call_location, caller_func, called_func, call.json_node);
                current_path.add_step(step);

                // Recurse into the called function
                dfs(call.called_function, current_path, visited);

                // Remove the step from the path (backtrack)
                current_path.steps.pop_back();
            }
        }

        visited.erase(current);
    };

    // Find all entry points (functions not called by anyone)
    std::set<std::string> all_functions;
    std::set<std::string> called_functions;

    for (const auto& call : function_calls) {
        all_functions.insert(call.caller_function);
        all_functions.insert(call.called_function);
        called_functions.insert(call.called_function);
    }

    // Also include standalone functions
    for (const auto& func : functions) {
        all_functions.insert(func.name);
    }

    // Start DFS from each entry point
    for (const auto& func : all_functions) {
        if (called_functions.find(func) == called_functions.end()) {
            CodePath path;
            std::set<std::string> visited;

            // For entry points, add their definition as the starting step
            const Function* entry_func = function_map[func];
            if (entry_func) {
                PathStep entry_step(entry_func->definition_location, nullptr, entry_func, entry_func->json_node);
                path.add_step(entry_step);
            }

            dfs(func, path, visited);
        }
    }

    return all_paths;
}

std::shared_ptr<Expression> AST::extract_argument_expression(const json* call_node, int arg_index) const
{
    if (!call_node) {
        logt << "extract_argument_expression: call_node is null";
        return nullptr;
    }
    if (!call_node->contains("inner")) {
        logt << "extract_argument_expression: no 'inner' field in call_node";
        return nullptr;
    }
    if (!(*call_node)["inner"].is_array()) {
        logt << "extract_argument_expression: 'inner' is not an array";
        return nullptr;
    }

    const auto& inner = (*call_node)["inner"];

    // Skip the first element (function reference) and get the argument at arg_index
    int current_arg = 0;
    for (size_t i = 1; i < inner.size(); ++i) {
        if (current_arg == arg_index) {
            const auto& arg_node = inner[i];
            return parse_expression(arg_node);
        }
        current_arg++;
    }

    return nullptr;
}

std::pair<bool, long> AST::extract_argument_value(const json* call_node, int arg_index) const
{
    if (!call_node || !call_node->contains("inner") || !(*call_node)["inner"].is_array()) {
        return {false, 0};
    }

    const auto& inner = (*call_node)["inner"];

    // Skip the first element (function reference) and get the argument at arg_index
    int current_arg = 0;
    for (size_t i = 1; i < inner.size(); ++i) {
        if (current_arg == arg_index) {
            const auto& arg_node = inner[i];

            // Parse the expression
            auto expr = parse_expression(arg_node);
            if (expr) {
                // Try to evaluate constant expressions
                auto evaluated_value = evaluate_expression(expr);
                if (evaluated_value.has_value()) {
                    return {true, evaluated_value.value()};
                }
            }

            // Fall back to old logic for now to maintain compatibility
            // Handle IntegerLiteral directly
            if (arg_node.value("kind", "") == "IntegerLiteral") {
                try {
                    return {true, std::stol(arg_node.value("value", "0"))};
                } catch (...) {
                    return {false, 0};
                }
            }

            // Handle ImplicitCastExpr (e.g., NullToPointer)
            if (arg_node.value("kind", "") == "ImplicitCastExpr" &&
                arg_node.contains("inner") && arg_node["inner"].is_array() &&
                !arg_node["inner"].empty()) {
                const auto& cast_inner = arg_node["inner"][0];
                if (cast_inner.value("kind", "") == "IntegerLiteral") {
                    try {
                        return {true, std::stol(cast_inner.value("value", "0"))};
                    } catch (...) {
                        return {false, 0};
                    }
                }
            }

            // Handle CStyleCastExpr (e.g., (void*)42)
            if (arg_node.value("kind", "") == "CStyleCastExpr" &&
                arg_node.contains("inner") && arg_node["inner"].is_array() &&
                !arg_node["inner"].empty()) {
                const auto& cast_inner = arg_node["inner"][0];
                if (cast_inner.value("kind", "") == "IntegerLiteral") {
                    try {
                        return {true, std::stol(cast_inner.value("value", "0"))};
                    } catch (...) {
                        return {false, 0};
                    }
                }
            }

            // Handle ParenExpr (e.g., (NULL) from macros)
            if (arg_node.value("kind", "") == "ParenExpr" &&
                arg_node.contains("inner") && arg_node["inner"].is_array() &&
                !arg_node["inner"].empty()) {
                // Parse the inner expression
                auto inner_expr = parse_expression(arg_node["inner"][0]);
                if (inner_expr && inner_expr->get_type() == Expression::Type::Literal) {
                    auto literal = std::dynamic_pointer_cast<LiteralExpression>(inner_expr);
                    if (literal) {
                        return {true, literal->get_value()};
                    }
                }
            }
        }
        current_arg++;
    }

    return {false, 0};
}

std::vector<CodePath> AST::find_call_paths_with_constraint(const std::string& target_function,
                                                          const std::string& arg_name,
                                                          long value) const
{
    // Get all paths to the target function
    auto all_paths = find_call_paths_to(target_function);
    std::vector<CodePath> filtered_paths;

    // For each path, generate an SMT-LIB model and check if it's satisfiable
    for (auto& path : all_paths) {
        // Build Model for this path
        Model model = build_model_for_path(path, target_function, arg_name, value);
        logd << "Checking model: " << model;

        // Check if the model is satisfiable using Z3
        try {
            bool is_sat = Z3Solver::check_sat(model);
            if (is_sat) {
                filtered_paths.push_back(path);
                logd << "Path is satisfiable";
            } else {
                logd << "Path is not satisfiable";
            }
        } catch (const std::exception& e) {
            logw << "Failed to check satisfiability: " << e.what();
        }
    }

    return filtered_paths;
}

std::shared_ptr<Expression> AST::parse_expression(const json& node) const
{
    if (!node.is_object()) {
        logt << "parse_expression: node is not an object";
        return nullptr;
    }

    std::string kind = node.value("kind", "");

    // Handle IntegerLiteral
    if (kind == "IntegerLiteral") {
        try {
            long value = std::stol(node.value("value", "0"));
            return std::make_shared<LiteralExpression>(value);
        } catch (...) {
            logt << "parse_expression: failed to parse IntegerLiteral value";
            return nullptr;
        }
    }

    // Handle BinaryOperator
    if (kind == "BinaryOperator") {
        std::string opcode = node.value("opcode", "");

        if (!node.contains("inner") || !node["inner"].is_array() || node["inner"].size() < 2) {
            logt << "parse_expression: BinaryOperator missing or invalid inner array";
            return nullptr;
        }

        auto left_expr = parse_expression(node["inner"][0]);
        auto right_expr = parse_expression(node["inner"][1]);

        if (!left_expr || !right_expr) {
            logt << "parse_expression: BinaryOperator failed to parse left or right expression";
            return nullptr;
        }

        if (opcode == "+") {
            auto result = std::make_shared<AdditionExpression>(left_expr, right_expr);
            return result;
        } else if (opcode == "-") {
            auto result = std::make_shared<SubtractionExpression>(left_expr, right_expr);
            return result;
        }
        // If we get here, it's a binary operator we don't handle
        logt << "parse_expression: unsupported binary operator: " << opcode;
        return nullptr;
    }

    // Handle UnaryOperator
    if (kind == "UnaryOperator") {
        std::string opcode = node.value("opcode", "");
        bool is_prefix = node.value("isPrefix", true);

        if (!node.contains("inner") || !node["inner"].is_array() || node["inner"].empty()) {
            logt << "parse_expression: UnaryOperator missing or invalid inner array";
            return nullptr;
        }

        auto operand = parse_expression(node["inner"][0]);
        if (!operand) {
            logt << "parse_expression: UnaryOperator failed to parse operand";
            return nullptr;
        }

        if (opcode == "-" && is_prefix) {
            return std::make_shared<NegationExpression>(operand);
        }

        logt << "parse_expression: unsupported unary operator: " << opcode;
    }

    // Handle DeclRefExpr (variable reference)
    if (kind == "DeclRefExpr") {
        if (node.contains("referencedDecl")) {
            auto ref_decl = node.value("referencedDecl", json::object());
            std::string ref_kind = ref_decl.value("kind", "");
            if (ref_kind == "ParmVarDecl" || ref_kind == "VarDecl") {
                std::string name = ref_decl.value("name", "");
                auto sv = std::make_shared<SymbolicVariable>(nullptr, name);
                return std::make_shared<VariableExpression>(sv);
            }
        }
    }

    // Handle other node types recursively (e.g., ParenExpr, CStyleCastExpr)
    if ((kind == "ParenExpr" || kind == "ImplicitCastExpr" || kind == "CStyleCastExpr") &&
        node.contains("inner") && node["inner"].is_array() && !node["inner"].empty()) {
        return parse_expression(node["inner"][0]);
    }

    logt << "parse_expression: unhandled node kind: " << kind << ", node" << node;
    return nullptr;
}

bool AST::check_path_satisfiability(const CodePath& path) const {
    if (path.get_constraints().empty()) {
        // No constraints means the path is always satisfiable
        return true;
    }

    Model model;

    // Add all constraints along the path to the model
    for (const auto& constraint : path.get_constraints()) {
        model.add_constraint(constraint);
    }

    // Check if the constraints are satisfiable
    return Z3Solver::check_sat(model);
}

std::string AST::generate_smtlib_for_path(const CodePath& path) const {
    if (path.get_constraints().empty()) {
        return "; No constraints on this path\n";
    }

    Model model;

    // Add all constraints along the path to the model
    for (const auto& constraint : path.get_constraints()) {
        model.add_constraint(constraint);
    }

    // Generate and return the SMT-LIB string
    return Z3Solver::to_smtlib(model);
}

std::optional<long> AST::evaluate_expression(const std::shared_ptr<Expression>& expr) const {
    if (!expr) {
        return std::nullopt;
    }

    switch (expr->get_type()) {
        case Expression::Type::Literal: {
            auto literal = std::dynamic_pointer_cast<LiteralExpression>(expr);
            if (literal) {
                return literal->get_value();
            }
            break;
        }

        case Expression::Type::Addition: {
            auto add_expr = std::dynamic_pointer_cast<AdditionExpression>(expr);
            if (add_expr) {
                auto left_val = evaluate_expression(add_expr->get_left());
                auto right_val = evaluate_expression(add_expr->get_right());
                if (left_val.has_value() && right_val.has_value()) {
                    return left_val.value() + right_val.value();
                }
            }
            break;
        }

        case Expression::Type::Subtraction: {
            auto sub_expr = std::dynamic_pointer_cast<SubtractionExpression>(expr);
            if (sub_expr) {
                auto left_val = evaluate_expression(sub_expr->get_left());
                auto right_val = evaluate_expression(sub_expr->get_right());
                if (left_val.has_value() && right_val.has_value()) {
                    return left_val.value() - right_val.value();
                }
            }
            break;
        }

        case Expression::Type::Negation: {
            auto neg_expr = std::dynamic_pointer_cast<NegationExpression>(expr);
            if (neg_expr) {
                auto operand_val = evaluate_expression(neg_expr->get_operand());
                if (operand_val.has_value()) {
                    return -operand_val.value();
                }
            }
            break;
        }

        default:
            break;
    }

    return std::nullopt;
}

// Helper to find a Variable by name in a function
const Variable* AST::find_variable(const std::string& function_name, const std::string& var_name) const
{
    // Find the function
    for (const auto& func : functions) {
        if (func.name == function_name) {
            // First, check the function's scope (includes both arguments and local variables)
            auto var = func.scope.lookup(var_name);
            if (var) {
                return var.get();
            }
        }
    }

    return nullptr;
}

Model AST::build_model_for_path(CodePath& path,
                                const std::string& target_function,
                                const std::string& arg_name,
                                long value) const
{
    Model model;

    std::stringstream ss;

    // Debug: print the path
    ss << "Path: ";
    for (const auto& step : path.steps) {
        ss << step.location.file << ":" << step.location.line << " ";
    }
    logd << ss.str();

    // Find the Variable for the target function's argument
    const Variable* target_var = find_variable(target_function, arg_name);

    // Start with the constraint on the target function's argument
    std::string initial_var_name = target_function + "_" + arg_name;
    auto initial_var = path.get_or_create_symbolic_variable(target_var, initial_var_name);

    // Add the initial constraint: foo_res = 0
    auto initial_constraint = std::make_shared<LinearEquation>(initial_var, value);
    model.add_constraint(initial_constraint);

    // Process constraints backwards through the path using new PathStep design
    if (!path.steps.empty()) {
        // Start from the last step (call to target function)
        int step_index = path.steps.size() - 1;
        process_variable_constraints(model, path, step_index, arg_name, target_function);
    }

    return model;
}

// Helper function to find all variables in an expression
std::set<std::string> AST::find_variables_in_expression(std::shared_ptr<Expression> expr) const
{
    std::set<std::string> vars;

    if (!expr) return vars;

    switch (expr->get_type()) {
        case Expression::Type::Variable: {
            auto var_expr = std::dynamic_pointer_cast<VariableExpression>(expr);
            if (var_expr) {
                vars.insert(var_expr->to_string());
            }
            break;
        }
        case Expression::Type::Addition: {
            auto add_expr = std::dynamic_pointer_cast<AdditionExpression>(expr);
            if (add_expr) {
                auto left_vars = find_variables_in_expression(add_expr->get_left());
                auto right_vars = find_variables_in_expression(add_expr->get_right());
                vars.insert(left_vars.begin(), left_vars.end());
                vars.insert(right_vars.begin(), right_vars.end());
            }
            break;
        }
        case Expression::Type::Subtraction: {
            auto sub_expr = std::dynamic_pointer_cast<SubtractionExpression>(expr);
            if (sub_expr) {
                auto left_vars = find_variables_in_expression(sub_expr->get_left());
                auto right_vars = find_variables_in_expression(sub_expr->get_right());
                vars.insert(left_vars.begin(), left_vars.end());
                vars.insert(right_vars.begin(), right_vars.end());
            }
            break;
        }
        case Expression::Type::Negation: {
            auto neg_expr = std::dynamic_pointer_cast<NegationExpression>(expr);
            if (neg_expr) {
                auto operand_vars = find_variables_in_expression(neg_expr->get_operand());
                vars.insert(operand_vars.begin(), operand_vars.end());
            }
            break;
        }
        case Expression::Type::Literal:
            // No variables in literals
            break;
    }

    return vars;
}

// Convert expression with variable prefix
std::shared_ptr<Expression> AST::convert_expression_with_prefix(std::shared_ptr<Expression> expr,
                                                               const std::string& var_prefix,
                                                               CodePath& path) const
{
    if (!expr) return nullptr;

    switch (expr->get_type()) {
        case Expression::Type::Literal: {
            // Literals don't need conversion
            return expr;
        }
        case Expression::Type::Variable: {
            auto var_expr = std::dynamic_pointer_cast<VariableExpression>(expr);
            if (var_expr) {
                // Get the variable name
                std::string var_name = var_expr->to_string();

                // Find the Variable in the prefix function
                std::string function_name = var_prefix.substr(0, var_prefix.length() - 1); // Remove trailing '_'
                const Variable* var = find_variable(function_name, var_name);

                // Create a new SymbolicVariable with the prefixed name
                std::string prefixed_name = var_prefix + var_name;
                auto sym_var = path.get_or_create_symbolic_variable(var, prefixed_name);
                return std::make_shared<VariableExpression>(sym_var);
            }
            break;
        }
        case Expression::Type::Addition: {
            auto add_expr = std::dynamic_pointer_cast<AdditionExpression>(expr);
            if (add_expr) {
                auto left = convert_expression_with_prefix(add_expr->get_left(), var_prefix, path);
                auto right = convert_expression_with_prefix(add_expr->get_right(), var_prefix, path);
                return std::make_shared<AdditionExpression>(left, right);
            }
            break;
        }
        case Expression::Type::Subtraction: {
            auto sub_expr = std::dynamic_pointer_cast<SubtractionExpression>(expr);
            if (sub_expr) {
                auto left = convert_expression_with_prefix(sub_expr->get_left(), var_prefix, path);
                auto right = convert_expression_with_prefix(sub_expr->get_right(), var_prefix, path);
                return std::make_shared<SubtractionExpression>(left, right);
            }
            break;
        }
        case Expression::Type::Negation: {
            auto neg_expr = std::dynamic_pointer_cast<NegationExpression>(expr);
            if (neg_expr) {
                auto operand = convert_expression_with_prefix(neg_expr->get_operand(), var_prefix, path);
                return std::make_shared<NegationExpression>(operand);
            }
            break;
        }
    }

    return nullptr;
}

void AST::process_variable_constraints(Model& model,
                                     CodePath& path,
                                     int step_index,
                                     const std::string& var_name,
                                     const std::string& function_name) const
{
    if (step_index < 0 || step_index >= static_cast<int>(path.steps.size())) {
        return;
    }

    // Get the PathStep at this index
    const PathStep& step = path.get_step(step_index);

    if (!step.called || step.called->name != function_name) {
        std::cerr << "Warning: Step does not match expected function call to " << function_name << std::endl;
        return;
    }

    // Log what we found
    logd << "Found call from " << (step.caller ? step.caller->name : "entry")
         << " to " << step.called->name
         << " at " << step.location.file << ":" << step.location.line;

    // Extract the argument expression from the called function's JSON node
    logt << "PathStep json_node pointer: " << step.call_json_node;
    assert(step.call_json_node && "All steps must have a JSON node");
    auto arg_expr = extract_argument_expression(step.call_json_node, 0);
    if (!arg_expr) {
        std::cerr << "Warning: Could not extract argument expression" << std::endl;
        return;
    }

    // Build constraint linking the target variable to the argument expression
    std::string caller_prefix = step.caller ? (step.caller->name + "_") : "";

    // Convert the expression with the caller's prefix
    auto converted_expr = convert_expression_with_prefix(arg_expr, caller_prefix, path);

    // Find the target Variable
    const Variable* target_variable = find_variable(function_name, var_name);

    // Create the target variable
    std::string target_var_name = function_name + "_" + var_name;
    auto target_var = path.get_or_create_symbolic_variable(target_variable, target_var_name);

    // Create the constraint: target_var = expression
    auto constraint = std::make_shared<LinearEquation>(target_var, converted_expr);
    model.add_constraint(constraint);

    // Find all variables in the expression
    std::set<std::string> vars_in_expr = find_variables_in_expression(arg_expr);

    // Recursively process variables in the expression
    for (const auto& var : vars_in_expr) {
        logt << "Processing variable: " << var << " in function " << function_name;
        
        // For variables in function call arguments, we need to look in the caller's scope
        std::string lookup_function = function_name;
        if (step.caller) {
            lookup_function = step.caller->name;
            logt << "Looking for variable in caller function: " << lookup_function;
        }
        
        // Check if this is a local variable with initialization
        const Variable* var_obj = find_variable(lookup_function, var);
        logt << "Found variable object: " << (var_obj ? "yes" : "no");
        if (var_obj && var_obj->get_kind() == Variable::Kind::LocalVariable && 
            var_obj->get_init_json_node()) {
            logt << "Variable " << var << " is local with initialization";
            // Parse the initialization expression
            auto init_expr = parse_expression(*var_obj->get_init_json_node());
            if (init_expr) {
                // Create symbolic variable for this local variable
                std::string local_var_name = lookup_function + "_" + var;
                auto local_var = path.get_or_create_symbolic_variable(var_obj, local_var_name);
                
                // Convert the initialization expression with proper prefix
                auto converted_init = convert_expression_with_prefix(init_expr, lookup_function + "_", path);
                
                // Add constraint: local_var = init_expr
                auto init_constraint = std::make_shared<LinearEquation>(local_var, converted_init);
                model.add_constraint(init_constraint);
                
                logd << "Added initialization constraint: " << local_var_name << " = " << init_expr->to_string();
            }
        } else if (step_index > 0 && step.caller) {
            // For function parameters or variables without initialization,
            // move to previous step in path and process that variable
            process_variable_constraints(model, path, step_index - 1, var, step.caller->name);
        }
    }
}

// Stream operators implementation
std::ostream& operator<<(std::ostream& os, const SourceLocation& location)
{
    os << location.file << ":" << location.line;
    if (location.column > 0) {
        os << ":" << location.column;
    }
    return os;
}

std::ostream& operator<<(std::ostream& os, const PathStep& step)
{
    os << step.location;
    if (step.caller) {
        os << " " << step.caller->name << " -> ";
    } else {
        os << " entry -> ";
    }
    if (step.called) {
        os << step.called->name;
    } else {
        os << "unknown";
    }
    return os;
}

std::ostream& operator<<(std::ostream& os, const CodePath& path)
{
    os << "CodePath (" << path.steps.size() << " steps";
    if (!path.constraints.empty()) {
        os << ", " << path.constraints.size() << " constraints";
    }
    os << "):\n";

    for (size_t i = 0; i < path.steps.size(); ++i) {
        os << "  [" << i << "] " << path.steps[i] << "\n";
    }

    if (!path.constraints.empty()) {
        os << "  Constraints:\n";
        for (size_t i = 0; i < path.constraints.size(); ++i) {
            os << "    [" << i << "] " << path.constraints[i]->to_string() << "\n";
        }
    }

    return os;
}
