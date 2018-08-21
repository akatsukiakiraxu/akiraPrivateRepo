#include <iostream>
#include <string>

int main(int argc, char const* argv[])
{
    for (std::string line; std::getline(std::cin, line);) {
        std::cerr << "RECIEVED: " << line << std::endl;
    }
    return 0;
}
