#include "ast-parser.h"
#include "log.h"
#include <fstream>
#include <iostream>
#include <memory>

AST AstParser::parse_ast_files(const std::filesystem::path& ast_directory)
{
    AST ast;

    if (!std::filesystem::exists(ast_directory)) {
        std::cerr << "AST directory does not exist: " << ast_directory << std::endl;
        return ast;
    }

    // Find all .ast.json files
    for (const auto& entry : std::filesystem::directory_iterator(ast_directory)) {
        if (entry.is_regular_file() && entry.path().extension() == ".json" &&
            entry.path().stem().extension() == ".ast") {

            logd << "Parsing AST file: " << entry.path();
            if (!parse_ast_file(entry.path(), ast)) {
                std::cerr << "Failed to parse AST file: " << entry.path() << std::endl;
            }
        }
    }

    return ast;
}

bool AstParser::parse_ast_file(const std::filesystem::path& ast_file, AST& ast)
{
    std::ifstream file(ast_file);
    if (!file.is_open()) {
        std::cerr << "Error: Cannot open AST file " << ast_file << std::endl;
        return false;
    }

    try {
        // Create a shared_ptr to store the JSON document
        auto ast_root_ptr = std::make_shared<json>();
        file >> *ast_root_ptr;

        // Store the JSON document in AST to keep pointers valid
        ast.add_json_document(ast_root_ptr);

        // Extract source filename from AST filename
        // e.g., "hello.ast.json" -> "hello.c"
        std::string source_file = ast_file.stem().stem().string() + ".c";

        // Process the AST tree starting from root
        process_node(*ast_root_ptr, ast, "", source_file);
        return true;
    } catch (const json::exception& e) {
        std::cerr << "Error parsing AST JSON in " << ast_file << ": " << e.what() << std::endl;
        return false;
    }
}

void AstParser::process_node(const json& node, AST& ast, const std::string& current_function,
                             const std::string& source_file)
{
    if (!node.is_object()) {
        return;
    }

    std::string kind = node.value("kind", "");
    std::string updated_function = current_function;

    // Check for function definitions
    if (kind == "FunctionDecl") {
        extract_function_definition(node, ast, source_file);
        // Update current function context for child nodes
        if (node.contains("name")) {
            updated_function = node["name"];
        }
    }

    // Check for function calls
    if (kind == "CallExpr") {
        extract_function_call(node, ast, current_function, source_file);
    }

    // Recursively process child nodes
    if (node.contains("inner") && node["inner"].is_array()) {
        for (const auto& child : node["inner"]) {
            process_node(child, ast, updated_function, source_file);
        }
    }
}

void AstParser::extract_function_definition(const json &node,
                                            AST &ast,
                                            const std::string &source_file)
{
    if (!node.contains("name")) {
        return;
    }

    std::string function_name = node["name"];
    SourceLocation location = extract_source_location(node, source_file);

    Function function(function_name, location);
    function.json_node = &node;  // Store pointer to JSON node

    // Extract return type if available
    if (node.contains("type") && node["type"].contains("qualType")) {
        function.return_type = node["type"]["qualType"];
    }

    // Extract arguments and create symbolic variables
    if (node.contains("inner") && node["inner"].is_array()) {
        int arg_index = 0;
        for (const auto& inner : node["inner"]) {
            if (inner.is_object() && inner.value("kind", "") == "ParmVarDecl") {
                std::string param_type = "";
                std::string param_name = "";
                int line_number = location.line;

                // Get parameter type
                if (inner.contains("type") && inner["type"].contains("qualType")) {
                    param_type = inner["type"]["qualType"];
                }

                // Get parameter name
                if (inner.contains("name")) {
                    param_name = inner["name"];
                } else {
                    // Generate a default name for unnamed parameters
                    param_name = "arg" + std::to_string(arg_index);
                }

                // Get line number if available
                if (inner.contains("range") && inner["range"].contains("begin") &&
                    inner["range"]["begin"].contains("line")) {
                    line_number = inner["range"]["begin"]["line"];
                }

                // Create shared_ptr<Variable> for this argument
                if (!param_type.empty()) {
                    // Create the variable with a pointer to the function
                    auto var = std::make_shared<Variable>(param_name, param_type, &function, arg_index,
                                                          source_file, line_number);
                    function.arguments.push_back(var);
                    function.scope.add_variable(var);
                    arg_index++;
                }
            }
        }
    }

    // Check if this is a definition or just a declaration
    // A function definition has a CompoundStmt as one of its inner nodes
    bool is_definition = false;
    if (node.contains("inner") && node["inner"].is_array()) {
        for (const auto& inner : node["inner"]) {
            if (inner.is_object() && inner.value("kind", "") == "CompoundStmt") {
                is_definition = true;
                // Process the function body to extract local variables
                extract_local_variables(inner, function, source_file);
                break;
            }
        }
    }

    function.defined = is_definition;
    ast.add_function(function);
}

void AstParser::extract_local_variables(const json& compound_stmt, Function& function,
                                        const std::string& source_file)
{
    if (!compound_stmt.contains("inner") || !compound_stmt["inner"].is_array()) {
        return;
    }

    // Process each statement in the compound statement
    for (const auto& stmt : compound_stmt["inner"]) {
        if (!stmt.is_object()) continue;

        std::string kind = stmt.value("kind", "");

        // Look for DeclStmt nodes which contain variable declarations
        if (kind == "DeclStmt" && stmt.contains("inner") && stmt["inner"].is_array()) {
            for (const auto& decl : stmt["inner"]) {
                if (decl.is_object() && decl.value("kind", "") == "VarDecl") {
                    std::string var_name = decl.value("name", "");
                    std::string var_type = "";

                    if (decl.contains("type") && decl["type"].contains("qualType")) {
                        var_type = decl["type"]["qualType"];
                    }

                    if (!var_name.empty() && !var_type.empty()) {
                        SourceLocation location = extract_source_location(decl, source_file);
                        
                        // Check if variable has initialization
                        const json* init_node = nullptr;
                        if (decl.contains("inner") && decl["inner"].is_array() && !decl["inner"].empty()) {
                            init_node = &decl["inner"][0];
                        }
                        
                        auto var = std::make_shared<Variable>(var_name, var_type, &function,
                                                            source_file, location.line, init_node);
                        function.scope.add_variable(var);
                    }
                }
            }
        }
        // Recursively process nested compound statements
        else if (kind == "CompoundStmt") {
            extract_local_variables(stmt, function, source_file);
        }
    }
}

void AstParser::extract_function_call(const json &node, AST &ast,
                                      const std::string &current_function,
                                      const std::string &source_file)
{
    if (current_function.empty()) {
        return;
    }

    std::string called_function = extract_function_name_from_call(node);
    if (called_function.empty()) {
        return;
    }

    SourceLocation location = extract_source_location(node, source_file);
    FunctionCall call(current_function, called_function, location);
    call.json_node = &node;  // Store pointer to JSON node
    logt << "Storing json_node pointer: " << call.json_node << " for call from " 
         << current_function << " to " << called_function;

    ast.add_function_call(call);
}

SourceLocation AstParser::extract_source_location(const json &location_node,
                                                  const std::string &source_file)
{
    SourceLocation location;

    // First try to get location from direct "loc" field
    if (location_node.contains("loc")) {
        const auto& loc = location_node["loc"];

        // Check if this location is from an included file
        if (loc.contains("includedFrom")) {
            // Mark as external/included file
            location.file = "elsewhere";
            location.line = 0;
            location.column = 0;
            return location;
        }

        location.file = loc.value("file", source_file);
        location.line = loc.value("line", 0);
        location.column = loc.value("col", 0);

        // If we found valid location info, return it
        if (location.line > 0) {
            return location;
        }
    }

    // Fallback to range.begin for call locations
    if (location_node.contains("range") && location_node["range"].contains("begin")) {
        const auto& begin = location_node["range"]["begin"];

        if (begin.contains("spellingLoc")) {
            const auto& spelling = begin["spellingLoc"];
            location.file = spelling.value("file", source_file);
            location.line = spelling.value("line", 0);
            location.column = spelling.value("col", 0);
        } else {
            location.file = begin.value("file", source_file);
            location.line = begin.value("line", 0);
            location.column = begin.value("col", 0);
        }
    }

    // Ensure we always have a file
    if (location.file.empty()) {
        location.file = source_file;
    }

    return location;
}

std::string AstParser::extract_function_name_from_call(const json& call_node)
{
    // Look for ImplicitCastExpr -> FunctionToPointerDecay -> DeclRefExpr pattern
    if (!call_node.contains("inner") || !call_node["inner"].is_array()) {
        return "";
    }

    for (const auto& child : call_node["inner"]) {
        if (!child.is_object()) continue;

        std::string child_kind = child.value("kind", "");

        // Look for ImplicitCastExpr with FunctionToPointerDecay
        if (child_kind == "ImplicitCastExpr" &&
            child.value("castKind", "") == "FunctionToPointerDecay") {

            if (child.contains("inner") && child["inner"].is_array()) {
                for (const auto& grandchild : child["inner"]) {
                    if (grandchild.value("kind", "") == "DeclRefExpr" &&
                        grandchild.contains("referencedDecl") &&
                        grandchild["referencedDecl"].contains("name")) {

                        return grandchild["referencedDecl"]["name"];
                    }
                }
            }
        }

        // Direct DeclRefExpr case
        if (child_kind == "DeclRefExpr" &&
            child.contains("referencedDecl") &&
            child["referencedDecl"].contains("name")) {

            return child["referencedDecl"]["name"];
        }
    }

    return "";
}
