#include "ast-generator.h"
#include "clang-executor.h"
#include <iostream>

bool AstGenerator::generate_ast_for_directory(const std::filesystem::path& source_dir,
                                              std::filesystem::path& ast_output_dir)
{
    std::filesystem::path compile_commands = source_dir / "compile_commands.json";

    if (!std::filesystem::exists(compile_commands)) {
        std::cerr << "Error: compile_commands.json not found in: " << source_dir << std::endl;
        return false;
    }

    ast_output_dir = source_dir / "ast-output";
    return generate_ast_from_compile_commands(compile_commands, ast_output_dir);
}

bool AstGenerator::generate_ast_from_compile_commands(const std::filesystem::path& compile_commands_file,
                                                      const std::filesystem::path& ast_output_dir)
{
    auto commands = CompileCommandsParser::parse(compile_commands_file);

    if (commands.empty()) {
        std::cerr << "Error: No compile commands found in " << compile_commands_file << std::endl;
        return false;
    }

    // Create AST output directory
    std::filesystem::create_directories(ast_output_dir);

    // Process each compile command
    bool all_success = true;
    for (const auto& command : commands) {
        if (!ClangExecutor::generate_ast(command, ast_output_dir)) {
            std::cerr << "Failed to generate AST for: " << command.file << std::endl;
            all_success = false;
        }
    }

    return all_success;
}