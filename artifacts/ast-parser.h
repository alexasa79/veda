#pragma once

#include "ast.h"
#include "json.hpp"
#include <filesystem>
#include <string>

using json = nlohmann::json;

class AstParser {
public:
    static AST parse_ast_files(const std::filesystem::path& ast_directory);
    static bool parse_ast_file(const std::filesystem::path& ast_file, AST& ast);

private:
    static void process_node(const json& node, AST& ast, const std::string& current_function, const std::string& source_file);
    static void extract_function_definition(const json& node, AST& ast, const std::string& source_file);
    static void extract_function_call(const json& node, AST& ast, const std::string& current_function, const std::string& source_file);
    static void extract_local_variables(const json& compound_stmt, Function& function, const std::string& source_file);
    static SourceLocation extract_source_location(const json& location_node, const std::string& source_file);
    static std::string extract_function_name_from_call(const json& call_node);
    static std::string find_current_function_context(const json& node);
};