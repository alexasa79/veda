CC = gcc
CFLAGS = -Wall -Wextra -g
TARGET = calculator
OBJS = main.o arithmetic.o advanced.o utils.o

all: $(TARGET)

$(TARGET): $(OBJS)
	$(CC) $(CFLAGS) -o $(TARGET) $(OBJS) -lm

main.o: main.c arithmetic.h advanced.h utils.h
	$(CC) $(CFLAGS) -c main.c

arithmetic.o: arithmetic.c arithmetic.h
	$(CC) $(CFLAGS) -c arithmetic.c

advanced.o: advanced.c advanced.h arithmetic.h
	$(CC) $(CFLAGS) -c advanced.c

utils.o: utils.c utils.h advanced.h
	$(CC) $(CFLAGS) -c utils.c

clean:
	rm -f $(OBJS) $(TARGET)

run: $(TARGET)
	./$(TARGET)

.PHONY: all clean run