#ifndef _Singleton_H_
#define _Singleton_H_

#include <boost/thread.hpp>
#include <memory>
#include <stdlib.h>
#include <stdio.h>

using namespace std;

template <typename T> class Singleton
{
public:
    static T& GetInstance()
    {
        static boost::mutex s_mutex;
        if (s_instance.get() == NULL)
        {
            boost::mutex::scoped_lock lock(s_mutex);
            if (s_instance.get() == NULL)
            {
               s_instance.reset(new T());
            }
            // 'lock' will be destructed now. 's_mutex' will be unlocked.
        }
        return *s_instance;
    }

protected:
    Singleton() { }
    ~Singleton() { }

    // Use auto_ptr to make sure that the allocated memory for instance
    // will be released when program exits (after main() ends).
    static std::unique_ptr<T> s_instance;

private:
    Singleton(const Singleton&);
    Singleton& operator =(const Singleton&);
};

template<typename T> std::unique_ptr<T> Singleton<T>::s_instance;

#endif /*_Singleton_H_*/