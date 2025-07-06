#include "compile-commands-parser.h"
#include <fstream>
#include <iostream>
#include "json.hpp"

using json = nlohmann::json;

std::vector<CompileCommand> CompileCommandsParser::parse(const std::filesystem::path& json_file)
{
    std::vector<CompileCommand> commands;

    std::ifstream file(json_file);
    if (!file.is_open()) {
        std::cerr << "Error: Cannot open " << json_file << std::endl;
        return commands;
    }

    try {
        json j;
        file >> j;

        if (!j.is_array()) {
            std::cerr << "Error: Expected JSON array in compile_commands.json" << std::endl;
            return commands;
        }

        // Get the directory containing compile_commands.json
        std::filesystem::path base_dir = json_file.parent_path();
        
        for (const auto& item : j) {
            CompileCommand command;

            if (item.contains("directory") && item["directory"].is_string()) {
                std::filesystem::path dir = item["directory"];
                // If directory is relative, resolve it relative to compile_commands.json location
                if (dir.is_relative()) {
                    // Check if it's a simple relative path like "artifacts/hello-syscall"
                    // In this case, the directory is meant to be relative to the project root
                    // not to the compile_commands.json location
                    if (dir.string() == base_dir.filename().string() || 
                        dir.string() == "artifacts/" + base_dir.filename().string()) {
                        // The directory is the same as where compile_commands.json is located
                        command.directory = base_dir.string();
                    } else {
                        command.directory = (base_dir / dir).string();
                    }
                } else {
                    command.directory = dir.string();
                }
            }

            if (item.contains("file") && item["file"].is_string()) {
                command.file = item["file"];
            }

            if (item.contains("arguments") && item["arguments"].is_array()) {
                for (const auto& arg : item["arguments"]) {
                    if (arg.is_string()) {
                        command.arguments.push_back(arg);
                    }
                }
            }

            if (!command.file.empty() && !command.arguments.empty()) {
                commands.push_back(command);
            }
        }
    } catch (const json::exception& e) {
        std::cerr << "Error parsing JSON: " << e.what() << std::endl;
    }

    return commands;
}

