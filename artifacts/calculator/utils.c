#include "utils.h"
#include "advanced.h"
#include <stdio.h>
#include <math.h>

const char* get_version(void) {
	return "1.0.0";
}

void log_operation(const char* op, double a, double b, double result) {
	printf("[LOG] %s(%.2f, %.2f) = %.2f\n", op, a, b, result);
}

bool is_zero(double value) {
	return absolute(value) < 0.000001;
}

bool is_prime(int n) {
	if (n <= 1) {
		return false;
	}
	if (n <= 3) {
		return true;
	}
	if (n % 2 == 0 || n % 3 == 0) {
		return false;
	}
	
	for (int i = 5; i * i <= n; i += 6) {
		if (n % i == 0 || n % (i + 2) == 0) {
			return false;
		}
	}
	
	return true;
}

void to_binary(int n, char* buffer) {
	if (n == 0) {
		buffer[0] = '0';
		buffer[1] = '\0';
		return;
	}
	
	int index = 0;
	int temp = n;
	
	if (n < 0) {
		n = -n;
	}
	
	char temp_buffer[32];
	while (n > 0) {
		temp_buffer[index++] = (n % 2) ? '1' : '0';
		n = n / 2;
	}
	
	int j = 0;
	if (temp < 0) {
		buffer[j++] = '-';
	}
	
	for (int i = index - 1; i >= 0; i--) {
		buffer[j++] = temp_buffer[i];
	}
	buffer[j] = '\0';
}

double max(double a, double b) {
	return a > b ? a : b;
}

double min(double a, double b) {
	return a < b ? a : b;
}

void swap(double* a, double* b) {
	double temp = *a;
	*a = *b;
	*b = temp;
}