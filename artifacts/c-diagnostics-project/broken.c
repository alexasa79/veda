#include <stdio.h>

int add(int a, int b) {
    int x = a + b // missing semicolon
    return x;
}

int main(void) {
    printf("%d\n", add(1, 2));
    return 0;
}
