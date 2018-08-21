/*
#include <iostream>
#include <fstream>
#include <string>
*/
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <io.h>
int main(int argc, char const* argv[])
{
    int c;
    //setmode(0, O_BINARY);
    //setmode(1, O_BINARY);
    while ((c = getchar()) != EOF) {
        putchar(c);
    }
/*
    for (std::string line; std::getline(std::cin, line);) {
        std::cerr << "RECIEVED: " << line << std::endl;
    }
*/
    return 0;
}
