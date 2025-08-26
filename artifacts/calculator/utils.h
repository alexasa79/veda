#ifndef UTILS_H
#define UTILS_H

#include <stdbool.h>

const char* get_version(void);
void log_operation(const char* op, double a, double b, double result);
bool is_zero(double value);
bool is_prime(int n);
void to_binary(int n, char* buffer);
double max(double a, double b);
double min(double a, double b);
void swap(double* a, double* b);

#endif