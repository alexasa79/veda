#pragma once

#include <filesystem>
#include <vector>
#include "compile-commands-parser.h"

class AstGenerator {
public:
    static bool generate_ast_for_directory(const std::filesystem::path& source_dir,
                                           std::filesystem::path& ast_output_dir);
    static bool generate_ast_from_compile_commands(const std::filesystem::path& compile_commands_file,
                                                   const std::filesystem::path& ast_output_dir);
};