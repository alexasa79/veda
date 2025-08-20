#ifndef     _GNU_SOURCE
  #define _GNU_SOURCE // for pthread_getaffinity_np()
#endif  // !_GNU_SOURCE

#include <stdio.h>

int main()
{
    printf("Hello world\n");
    return 0;
}