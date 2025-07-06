#include <iostream>
#include <filesystem>
#include <string>
#include <map>
#include <sstream>
#include "linenoise.h"
#include "compile-commands-parser.h"
#include "clang-executor.h"
#include "ast-parser.h"
#include "ast-generator.h"
#include "ast.h"
#include "utils.h"
#include "log.h"

void print_usage(const char* program_name)
{
    std::cout << "Usage: " << program_name << " [options] <directory>" << std::endl;
    std::cout << "  directory: Path to directory containing compile_commands.json" << std::endl;
    std::cout << std::endl;
    std::cout << "Options:" << std::endl;
    std::cout << "  -d, --debug         Set log level to debug (creates log file)" << std::endl;
    std::cout << "  -t, --trace         Set log level to trace (creates log file)" << std::endl;
    std::cout << "  --log-file <file>   Specify custom log file name" << std::endl;
    std::cout << "  -h, --help          Show this help message" << std::endl;
}

// Command handler functions
void handle_help_command()
{
    std::cout << "Available commands:" << std::endl;
    std::cout << "  functions                                  - List all functions" << std::endl;
    std::cout << "  signature <func>                           - Show signature of a specific function" << std::endl;
    std::cout << "  codepath to <func>                         - Show all call paths to a function" << std::endl;
    std::cout << "  codepath to <func> where <arg> is <value>  - Show paths where argument has specific value" << std::endl;
    std::cout << "  exit                                       - Exit the program" << std::endl;
}

void handle_functions_command(const AST& ast)
{
    const auto& functions = ast.get_functions();
    int defined_count = 0;
    for (const auto& func : functions) {
        if (func.defined) defined_count++;
    }

    std::cout << "Functions in the project (" << defined_count << " defined):" << std::endl;
    for (const auto& func : functions) {
        if (func.defined) {
            std::cout << "  " << func.name;
            if (func.definition_location.file == "elsewhere") {
                std::cout << " (defined elsewhere)";
            } else {
                std::cout << " (" << func.definition_location.file << ":"
                          << func.definition_location.line << ")";
            }
            std::cout << std::endl;
        }
    }
}

void handle_signature_command(const AST& ast, const std::string& target_function)
{
    const auto& functions = ast.get_functions();
    bool found = false;

    for (const auto& func : functions) {
        if (func.name == target_function) {
            found = true;

            // Display return type and function name
            if (!func.return_type.empty()) {
                // Extract just the return type from the full function type
                size_t paren_pos = func.return_type.find('(');
                if (paren_pos != std::string::npos) {
                    std::cout << func.return_type.substr(0, paren_pos - 1) << " ";
                } else {
                    std::cout << func.return_type << " ";
                }
            }
            std::cout << func.name << "(";

            // Display arguments
            for (size_t i = 0; i < func.arguments.size(); ++i) {
                if (i > 0) std::cout << ", ";
                const auto& arg = func.arguments[i];
                std::cout << arg->get_type() << " " << arg->get_name();
            }

            std::cout << ")";

            // Show if it's external
            if (!func.defined) {
                std::cout << " [external]";
            }

            std::cout << " at " << func.definition_location.file << ":"
                      << func.definition_location.line << std::endl;
            break;
        }
    }

    if (!found) {
        std::cout << "Function not found: " << target_function << std::endl;
    }
}

void handle_codepath_command(const AST& ast, const std::string& target_function)
{
    auto paths = ast.find_call_paths_to(target_function);

    if (paths.empty()) {
        std::cout << "No call paths found to function: " << target_function << std::endl;
    } else {
        std::cout << "Call paths to " << target_function << " (" << paths.size() << " found):" << std::endl;

        // Build maps for looking up function names
        std::map<SourceLocation, std::string> def_location_to_function;
        for (const auto& func : ast.get_functions()) {
            def_location_to_function[func.definition_location] = func.name;
        }

        std::map<SourceLocation, std::string> call_location_to_called_function;
        for (const auto& call : ast.get_function_calls()) {
            call_location_to_called_function[call.call_location] = call.called_function;
        }

        for (size_t i = 0; i < paths.size(); ++i) {
            std::cout << "  Path " << (i + 1) << ": ";
            for (size_t j = 0; j < paths[i].steps.size(); ++j) {
                if (j > 0) std::cout << " -> ";
                const auto& step = paths[i].steps[j];

                // With PathStep, we have direct access to function names
                std::string func_name = step.called ? step.called->name : "unknown";
                bool is_entry_point = (step.caller == nullptr);

                // Only show location for actual calls, not entry points
                if (is_entry_point && j == 0) {
                    std::cout << func_name;
                } else {
                    std::cout << func_name << " (" << step.location.file << ":" << step.location.line << ")";
                }
            }
            std::cout << std::endl;
        }
    }
}

void handle_codepath_with_constraint_command(const AST& ast, const std::string& target_function,
                                             const std::string& arg_name, const std::string& value_str)
{
    // Convert value string to long
    long value;
    try {
        value = std::stol(value_str);
    } catch (const std::exception& e) {
        std::cout << "Invalid value: " << value_str << " (must be a number)" << std::endl;
        return;
    }

    auto paths = ast.find_call_paths_with_constraint(target_function, arg_name, value);

    if (paths.empty()) {
        std::cout << "No call paths found to function " << target_function
                  << " where " << arg_name << " is " << value << std::endl;
        return;
    }

    std::cout << "Call paths to " << target_function
              << " where " << arg_name << " is " << value
              << " (" << paths.size() << " found):" << std::endl;

    // Build maps for looking up function names
    std::map<SourceLocation, std::string> def_location_to_function;
    for (const auto& func : ast.get_functions()) {
        def_location_to_function[func.definition_location] = func.name;
    }

    std::map<SourceLocation, std::string> call_location_to_called_function;
    for (const auto& call : ast.get_function_calls()) {
        call_location_to_called_function[call.call_location] = call.called_function;
    }

    for (size_t i = 0; i < paths.size(); ++i) {
        std::cout << "  Path " << (i + 1) << ": ";
        for (size_t j = 0; j < paths[i].steps.size(); ++j) {
            if (j > 0) std::cout << " -> ";
            const auto& step = paths[i].steps[j];

            // With PathStep, we have direct access to function names
            std::string func_name = step.called ? step.called->name : "unknown";
            bool is_entry_point = (step.caller == nullptr);

            // Only show location for actual calls, not entry points
            if (is_entry_point && j == 0) {
                std::cout << func_name;
            } else {
                std::cout << func_name << " (" << step.location.file << ":" << step.location.line << ")";
            }
        }
        std::cout << std::endl;

        // Check path satisfiability and show SMT-LIB
        logd << "  Constraints satisfiable: "
             << (ast.check_path_satisfiability(paths[i]) ? "YES" : "NO");
        logd << "  SMT-LIB representation:";
        std::string smtlib = ast.generate_smtlib_for_path(paths[i]);
        std::istringstream iss(smtlib);
        std::string line;
        while (std::getline(iss, line)) {
            logd << "    " << line;
        }
    }
}

void run_interactive_mode(const AST& ast)
{
    std::cout << "Interactive mode started. Type 'help' for available commands." << std::endl;

    char* input;
    while ((input = linenoise("shleuth> ")) != nullptr) {
        // Add to history if not empty
        if (*input) {
            linenoiseHistoryAdd(input);
        }

        std::string command(input);
        free(input);

        // Parse command into words
        auto words = split_command(command);
        if (words.empty()) continue;

        if (words[0] == "help") {
            handle_help_command();
        } else if (words[0] == "functions") {
            handle_functions_command(ast);
        } else if (words.size() >= 2 && words[0] == "signature") {
            handle_signature_command(ast, words[1]);
        } else if (words.size() >= 3 && words[0] == "codepath" && words[1] == "to") {
            // Check for extended syntax: codepath to FUNC where ARG is VALUE
            if (words.size() >= 7 && words[3] == "where" && words[5] == "is") {
                handle_codepath_with_constraint_command(ast, words[2], words[4], words[6]);
            } else {
                handle_codepath_command(ast, words[2]);
            }
        } else if (words[0] == "exit") {
            break;
        } else {
            std::cout << "Unknown command: " << command << std::endl;
        }
    }

    std::cout << "Goodbye!" << std::endl;
}

int main(int argc, char* argv[])
{
    std::cout << "Shleuth v1.0.0" << std::endl;

    // Parse command line arguments
    LogLevel log_level = LogLevel::Info;
    std::string log_file;
    std::string directory;

    int i = 1;
    while (i < argc) {
        std::string arg = argv[i];

        if (arg == "-d" || arg == "--debug") {
            log_level = LogLevel::Debug;
            i++;
        } else if (arg == "-t" || arg == "--trace") {
            log_level = LogLevel::Trace;
            i++;
        } else if (arg == "--log-file") {
            if (i + 1 >= argc) {
                std::cerr << "Error: --log-file requires an argument" << std::endl;
                return 1;
            }
            log_file = argv[i + 1];
            i += 2;
        } else if (arg == "-h" || arg == "--help") {
            print_usage(argv[0]);
            return 0;
        } else if (arg[0] == '-') {
            std::cerr << "Error: Unknown option: " << arg << std::endl;
            print_usage(argv[0]);
            return 1;
        } else {
            // This must be the directory
            directory = arg;
            i++;
            break;
        }
    }

    if (directory.empty()) {
        std::cerr << "Error: No directory specified" << std::endl;
        print_usage(argv[0]);
        return 1;
    }

    // Initialize logging with parsed options
    Log::initialize(argv[0], log_level, log_file);
    logi << "Code analyzer started";

    std::filesystem::path source_dir(directory);

    if (!std::filesystem::exists(source_dir)) {
        std::cerr << "Error: Directory does not exist: " << source_dir << std::endl;
        return 1;
    }

    if (!std::filesystem::is_directory(source_dir)) {
        std::cerr << "Error: Path is not a directory: " << source_dir << std::endl;
        return 1;
    }

    logd << "Parsing compile_commands.json...";

    // Generate AST files using the new module
    std::filesystem::path ast_dir;
    bool all_success = AstGenerator::generate_ast_for_directory(source_dir, ast_dir);

    if (all_success) {
        logd << "All AST files generated successfully in: " << ast_dir;

        // Parse AST files
        AST ast = AstParser::parse_ast_files(ast_dir);

        // Start interactive mode
        run_interactive_mode(ast);

    } else {
        logw << "Some AST generation failed. Check error messages above.";
        return 1;
    }

    return 0;
}
