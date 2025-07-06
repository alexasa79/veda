#include "clang-executor.h"
#include <iostream>
#include <sstream>
#include <cstdlib>
#include "log.h"

bool ClangExecutor::generate_ast(const CompileCommand& command,
                                 const std::filesystem::path& output_dir)
{
    // Create output directory if it doesn't exist
    std::filesystem::create_directories(output_dir);

    // Get the AST output paths
    std::filesystem::path ast_json_output = get_ast_output_path(command, output_dir, ".ast.json");
    std::filesystem::path ast_text_output = get_ast_output_path(command, output_dir, ".ast.txt");

    // Build and execute JSON AST command
    std::string json_cmd = build_clang_command(command, ast_json_output, true);
    logd << "Executing: " << json_cmd;

    int json_result = std::system(json_cmd.c_str());
    if (json_result != 0) {
        std::cerr << "Error: JSON AST command failed with exit code " << json_result << std::endl;
        return false;
    }

    // Build and execute human-readable AST command
    std::string text_cmd = build_clang_command(command, ast_text_output, false);
    logd << "Executing: " << text_cmd;

    int text_result = std::system(text_cmd.c_str());
    if (text_result != 0) {
        std::cerr << "Error: Human-readable AST command failed with exit code " << text_result
                  << std::endl;
        return false;
    }

    logd << "AST generated: " << ast_json_output << " and " << ast_text_output;
    return true;
}

std::string ClangExecutor::build_clang_command(const CompileCommand& command,
                                               const std::filesystem::path& ast_output,
                                               bool json_format)
{
    std::ostringstream cmd;

    // Start with clang
    cmd << "clang";

    // Add original arguments, but skip the output-related ones and add --emit-ast
    bool skip_next = false;
    for (size_t i = 0; i < command.arguments.size(); ++i) {
        const std::string& arg = command.arguments[i];

        if (skip_next) {
            skip_next = false;
            continue;
        }

        // Skip compiler name (first argument)
        if (i == 0) continue;

        // Skip output-related arguments
        if (arg == "-o" || arg == "--output") {
            skip_next = true;
            continue;
        }

        // Skip existing AST dump flags and output flags
        if (arg.find("-ast-") != std::string::npos) {
            continue;
        }

        cmd << " \"" << arg << "\"";
    }

    // Add AST dump flag based on format
    if (json_format) {
        cmd << " -Xclang -ast-dump=json";
    } else {
        cmd << " -Xclang -ast-dump";
    }

    // Redirect output to file
    cmd << " > \"" << ast_output.string() << "\"";

    // Change to the correct directory
    std::string full_cmd = "cd \"" + command.directory + "\" && " + cmd.str();

    return full_cmd;
}

std::filesystem::path ClangExecutor::get_ast_output_path(const CompileCommand& command,
                                                         const std::filesystem::path& output_dir,
                                                         const std::string& extension)
{
    std::filesystem::path source_file(command.file);
    std::string filename = source_file.stem().string() + extension;
    // Return path relative to the working directory (command.directory)
    std::filesystem::path relative_output = std::filesystem::relative(output_dir, command.directory);
    return relative_output / filename;
}