#include "log.h"
#include <iostream>
#include <iomanip>
#include <chrono>
#include <ctime>
#include <filesystem>
#include <cstdlib>

// Initialize static members
LogLevel Log::current_level_ = LogLevel::Info;
std::unique_ptr<std::ofstream> Log::log_file_ = nullptr;
std::string Log::log_filename_;

// Helper class for automatic cleanup
class LogCleanup {
public:
    ~LogCleanup() {
        if (Log::log_file_ && Log::log_file_->is_open()) {
            // Write a shutdown message before closing
            auto now = std::chrono::system_clock::now();
            auto time_t_now = std::chrono::system_clock::to_time_t(now);
            auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                now.time_since_epoch()) % 1000;
            
            std::tm tm_now;
            localtime_r(&time_t_now, &tm_now);
            
            std::ostringstream oss;
            oss << "[" << std::put_time(&tm_now, "%Y-%m-%d %H:%M:%S") 
                << "." << std::setfill('0') << std::setw(3) << ms.count() << "] "
                << "[INFO ] [log.cpp:" << __LINE__ << "] [~LogCleanup] "
                << "Log system shutting down";
            
            *Log::log_file_ << oss.str() << std::endl;
        }
        Log::shutdown();
    }
};

// Static instance to ensure cleanup on exit
static LogCleanup log_cleanup;

// Helper function to get log level string
static const char* level_to_string(LogLevel level) {
    switch (level) {
        case LogLevel::Fatal:   return "FATAL";
        case LogLevel::Warning: return "WARN ";
        case LogLevel::Info:    return "INFO ";
        case LogLevel::Debug:   return "DEBUG";
        case LogLevel::Trace:   return "TRACE";
    }
    return "?????";
}

// Helper function to get basename from full path
static std::string get_basename(const std::string& path) {
    size_t pos = path.find_last_of("/\\");
    return (pos == std::string::npos) ? path : path.substr(pos + 1);
}

LogEntry::LogEntry(LogLevel level, const std::string& file, int line, const std::string& function)
    : level_(level), file_(file), line_(line), function_(function) {
}

LogEntry::~LogEntry() {
    // Only output if this log level is enabled
    if (!Log::is_enabled(level_)) {
        return;
    }
    
    // Get the raw message
    std::string raw_message = buffer_.str();
    
    // For info, warning, and fatal: always output to stderr (raw message only)
    if (level_ <= LogLevel::Info) {
        std::cerr << raw_message << std::endl;
    }
    
    // Write to log file if it's open (with full metadata)
    if (Log::log_file_ && Log::log_file_->is_open()) {
        // Get current timestamp
        auto now = std::chrono::system_clock::now();
        auto time_t_now = std::chrono::system_clock::to_time_t(now);
        auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(
            now.time_since_epoch()) % 1000;
        
        // Format timestamp
        std::tm tm_now;
        localtime_r(&time_t_now, &tm_now);
        
        // Output format: [TIMESTAMP] [LEVEL] [FILE:LINE] [FUNCTION] MESSAGE
        std::ostringstream oss;
        oss << "[" << std::put_time(&tm_now, "%Y-%m-%d %H:%M:%S") 
            << "." << std::setfill('0') << std::setw(3) << ms.count() << "] "
            << "[" << level_to_string(level_) << "] "
            << "[" << get_basename(file_) << ":" << line_ << "] "
            << "[" << function_ << "] "
            << raw_message;
        
        *Log::log_file_ << oss.str() << std::endl;
        Log::log_file_->flush(); // Ensure it's written immediately
    } else if (level_ > LogLevel::Info) {
        // For debug and trace, output to stderr only if no file is available
        std::cerr << raw_message << std::endl;
    }
    
    // Fatal messages terminate the program
    if (level_ == LogLevel::Fatal) {
        std::abort();
    }
}

void Log::set_level(LogLevel level) {
    current_level_ = level;
}

LogLevel Log::get_level() {
    return current_level_;
}

bool Log::is_enabled(LogLevel level) {
    return static_cast<int>(level) <= static_cast<int>(current_level_);
}

LogEntry Log::create_entry(LogLevel level, const std::string& file, 
                           int line, const std::string& function) {
    return LogEntry(level, file, line, function);
}

void Log::initialize(const std::string& argv0) {
    initialize(argv0, current_level_, "");
}

void Log::initialize(const std::string& argv0, LogLevel level, 
                     const std::string& custom_filename) {
    current_level_ = level;
    
    // Only create log file if log level is Debug or Trace
    if (level >= LogLevel::Debug) {
        if (!custom_filename.empty()) {
            log_filename_ = custom_filename;
        } else {
            // Extract executable name from argv[0]
            std::filesystem::path exe_path(argv0);
            std::string exe_name = exe_path.filename().string();
            log_filename_ = exe_name + ".log";
        }
        
        // Open log file
        log_file_ = std::make_unique<std::ofstream>(log_filename_, std::ios::app);
        if (!log_file_->is_open()) {
            std::cerr << "Warning: Failed to open log file: " << log_filename_ << std::endl;
        }
    }
}

void Log::shutdown() {
    if (log_file_ && log_file_->is_open()) {
        log_file_->flush();
        log_file_->close();
    }
    log_file_.reset();
}

