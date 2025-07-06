#pragma once

#include <string>
#include <vector>
#include <filesystem>

struct CompileCommand {
    std::string directory;
    std::string file;
    std::vector<std::string> arguments;
};

class CompileCommandsParser {
public:
    static std::vector<CompileCommand> parse(const std::filesystem::path& json_file);
};