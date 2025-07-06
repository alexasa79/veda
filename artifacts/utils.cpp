#include "utils.h"
#include <sstream>
#include <algorithm>

std::vector<std::string> split_command(const std::string& command)
{
    std::vector<std::string> words;
    std::istringstream stream(command);
    std::string word;
    
    // Split by whitespace
    while (stream >> word) {
        // Trim any remaining whitespace (though >> operator already does this)
        word.erase(0, word.find_first_not_of(" \t\n\r"));
        word.erase(word.find_last_not_of(" \t\n\r") + 1);
        
        if (!word.empty()) {
            words.push_back(word);
        }
    }
    
    return words;
}