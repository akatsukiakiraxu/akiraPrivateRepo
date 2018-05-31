.PHONY: all clean


all: sample

clean:
	-@rm sample test

sample: sample.c
	gcc -o $@ $<

test:
	dd if=/dev/urandom of=test bs=1024 count=2048

run: sample test
	./sample test 127.0.0.1 80

