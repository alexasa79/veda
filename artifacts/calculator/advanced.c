#include "advanced.h"
#include "arithmetic.h"

double power(double base, int exponent) {
	if (exponent == 0) {
		return 1.0;
	}
	
	double result = 1.0;
	int abs_exp = exponent < 0 ? -exponent : exponent;
	
	for (int i = 0; i < abs_exp; i++) {
		result = multiply(result, base);
	}
	
	if (exponent < 0) {
		result = divide(1.0, result);
	}
	
	return result;
}

long factorial(int n) {
	if (n <= 1) {
		return 1;
	}
	return n * factorial(n - 1);
}

int gcd(int a, int b) {
	if (b == 0) {
		return a;
	}
	return gcd(b, a % b);
}

int lcm(int a, int b) {
	return (a * b) / gcd(a, b);
}

double square_root(double n) {
	if (n < 0) {
		return -1;
	}
	
	double x = n;
	double root;
	
	while (1) {
		root = 0.5 * (x + divide(n, x));
		
		double diff = x - root;
		if (diff < 0) diff = -diff;
		
		if (diff < 0.00001) {
			break;
		}
		
		x = root;
	}
	
	return root;
}

double absolute(double n) {
	return n < 0 ? subtract(0, n) : n;
}