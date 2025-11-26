#include <stdio.h>

int add(int a, int b) {
	return a + b;
}

int main(void) {
int value = add(1, 2);
	printf("%d\n", value);
	return 0;
}
