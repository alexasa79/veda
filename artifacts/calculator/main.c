#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "arithmetic.h"
#include "advanced.h"
#include "utils.h"

void print_menu() {
	printf("\n=== Calculator Menu ===\n");
	printf("1. Add two numbers\n");
	printf("2. Subtract two numbers\n");
	printf("3. Multiply two numbers\n");
	printf("4. Divide two numbers\n");
	printf("5. Calculate power\n");
	printf("6. Calculate factorial\n");
	printf("7. Calculate GCD\n");
	printf("8. Check if prime\n");
	printf("9. Convert to binary\n");
	printf("0. Exit\n");
	printf("Choice: ");
}

int main(int argc, char *argv[]) {
	int choice;
	double a, b, result;
	int n, m;
	char binary[33];
	
	printf("Simple Calculator\n");
	printf("Version: %s\n", get_version());
	
	while (1) {
		print_menu();
		scanf("%d", &choice);
		
		switch (choice) {
			case 1:
				printf("Enter two numbers: ");
				scanf("%lf %lf", &a, &b);
				result = add(a, b);
				printf("Result: %.2f\n", result);
				log_operation("add", a, b, result);
				break;
				
			case 2:
				printf("Enter two numbers: ");
				scanf("%lf %lf", &a, &b);
				result = subtract(a, b);
				printf("Result: %.2f\n", result);
				log_operation("subtract", a, b, result);
				break;
				
			case 3:
				printf("Enter two numbers: ");
				scanf("%lf %lf", &a, &b);
				result = multiply(a, b);
				printf("Result: %.2f\n", result);
				log_operation("multiply", a, b, result);
				break;
				
			case 4:
				printf("Enter two numbers: ");
				scanf("%lf %lf", &a, &b);
				if (is_zero(b)) {
					printf("Error: Division by zero!\n");
				} else {
					result = divide(a, b);
					printf("Result: %.2f\n", result);
					log_operation("divide", a, b, result);
				}
				break;
				
			case 5:
				printf("Enter base and exponent: ");
				scanf("%lf %d", &a, &n);
				result = power(a, n);
				printf("Result: %.2f\n", result);
				break;
				
			case 6:
				printf("Enter a non-negative integer: ");
				scanf("%d", &n);
				if (n < 0) {
					printf("Error: Factorial of negative number!\n");
				} else {
					printf("Result: %ld\n", factorial(n));
				}
				break;
				
			case 7:
				printf("Enter two integers: ");
				scanf("%d %d", &n, &m);
				printf("GCD: %d\n", gcd(n, m));
				break;
				
			case 8:
				printf("Enter an integer: ");
				scanf("%d", &n);
				if (is_prime(n)) {
					printf("%d is prime\n", n);
				} else {
					printf("%d is not prime\n", n);
				}
				break;
				
			case 9:
				printf("Enter an integer: ");
				scanf("%d", &n);
				to_binary(n, binary);
				printf("Binary: %s\n", binary);
				break;
				
			case 0:
				printf("Goodbye!\n");
				exit(0);
				
			default:
				printf("Invalid choice!\n");
		}
	}
	
	return 0;
}