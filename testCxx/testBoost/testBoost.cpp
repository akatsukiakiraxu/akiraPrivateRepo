//
// Created by xiaoming.xu on 2018/07/12.
//
#include <boost/property_tree/ptree.hpp>
#include <boost/property_tree/json_parser.hpp>
#include <boost/lexical_cast.hpp>
using boost::property_tree::ptree;

int main() {
    std::string ss = "{ \"id\" : \"123\", \"number\" : \"456\", \"stuff\" : [{ \"111\" : \"1024\" }, { \"222\" : \"2048\" }, { \"333\" : \"3072\" }] }";

    ptree pt;
    std::istringstream is(ss);
    read_json(is, pt);

    std::cout << "id:     " << pt.get<std::string>("id") << "\n";
    std::cout << "number: " << pt.get<std::string>("number") << "\n";
    for (auto& e : pt.get_child("stuff")) {
        for (const auto& kv : e.second) {
            std::cout << "key = " << kv.first << std::endl;
            if (boost::lexical_cast<int>(kv.first) == 111) {
                std::cout << "OK" << std::endl;
            }
            std::cout << "val = " << kv.second.get_value<int>() << std::endl;
        }
    }
}
