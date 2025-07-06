#pragma once

#include "compile-commands-parser.h"
#include <filesystem>

class ClangExecutor {
public:
    static bool generate_ast(const CompileCommand& command,
                             const std::filesystem::path& output_dir);

private:
    static std::string build_clang_command(const CompileCommand& command,
                                           const std::filesystem::path& ast_output, bool json_format);
    static std::filesystem::path get_ast_output_path(const CompileCommand& command,
                                                     const std::filesystem::path& output_dir,
                                                     const std::string& extension);
};