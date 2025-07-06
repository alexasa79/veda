#ifndef LOG_H
#define LOG_H

#include <iostream>
#include <sstream>
#include <string>
#include <fstream>
#include <memory>

enum class LogLevel {
    Fatal = 0,
    Warning = 1,
    Info = 2,
    Debug = 3,
    Trace = 4
};

class LogEntry {
public:
    LogEntry(LogLevel level, const std::string& file, int line, const std::string& function);
    ~LogEntry();
    
    template<typename T>
    LogEntry& operator<<(const T& value) {
        buffer_ << value;
        return *this;
    }
    
    // Handle stream manipulators like std::endl
    LogEntry& operator<<(std::ostream& (*manip)(std::ostream&)) {
        buffer_ << manip;
        return *this;
    }
    
private:
    LogLevel level_;
    std::string file_;
    int line_;
    std::string function_;
    std::ostringstream buffer_;
};

class Log {
public:
    static void initialize(const std::string& argv0);
    static void initialize(const std::string& argv0, LogLevel level, 
                          const std::string& custom_filename = "");
    static void shutdown();
    
    static void set_level(LogLevel level);
    static LogLevel get_level();
    static bool is_enabled(LogLevel level);
    
    static LogEntry create_entry(LogLevel level, const std::string& file, 
                                  int line, const std::string& function);
    
private:
    friend class LogEntry;
    friend class LogCleanup;
    static LogLevel current_level_;
    static std::unique_ptr<std::ofstream> log_file_;
    static std::string log_filename_;
};

// Macro definitions for logging at different levels
#define logf Log::create_entry(LogLevel::Fatal, __FILE__, __LINE__, __FUNCTION__)
#define logw Log::create_entry(LogLevel::Warning, __FILE__, __LINE__, __FUNCTION__)
#define logi Log::create_entry(LogLevel::Info, __FILE__, __LINE__, __FUNCTION__)
#define logd Log::create_entry(LogLevel::Debug, __FILE__, __LINE__, __FUNCTION__)
#define logt Log::create_entry(LogLevel::Trace, __FILE__, __LINE__, __FUNCTION__)

#endif // LOG_H