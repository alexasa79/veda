int f(int arg1)
{
    return arg1;
}

int add(int a, int b) 
{
    f();    
    int x = a + b // missing semicolon
    return x;
}

int main(void) {
    printf("%d\n", add(1, 2));
    return 0;
}
