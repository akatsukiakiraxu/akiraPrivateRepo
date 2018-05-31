#ifndef _ASYNCCONTROL_IMPL_HPP_
#define _ASYNCCONTROL_IMPL_HPP_

#include <vector>
#include <boost/thread.hpp>

#include "Singleton.hpp"
#include "AsyncCommand.hpp"


class AsyncControlImpl : public Singleton<AsyncControlImpl>
{
public:
    AsyncControlImpl();
    virtual ~AsyncControlImpl();
    AsyncControlImpl(const AsyncControlImpl&);
private:
    friend Singleton<AsyncControlImpl>;
    friend class std::unique_ptr<AsyncControlImpl>;

    AsyncControlImpl& operator =(const AsyncControlImpl&);
    inline AsyncControlImpl& getInstance(void)
    {
        return Singleton<AsyncControlImpl>::GetInstance();
    }

    list<AsyncCommand*> cmdque;
    boost::mutex m_mutex;
    bool m_isInit;

public:
    void addCmd(AsyncCommand* cmd);
    AsyncCommand* getCmd();
};
#endif // _ASYNCCONTROL_HPP_