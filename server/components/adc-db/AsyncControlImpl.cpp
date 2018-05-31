#include <stdlib.h>
#include "AsyncControlImpl.hpp"
#include "PropertyUtility.hpp"
#include "debug.h"

AsyncControlImpl::AsyncControlImpl()
{
    if (!m_isInit) {
        m_isInit = true;
    }
}

AsyncControlImpl::~AsyncControlImpl()
{
}
void AsyncControlImpl::addCmd(AsyncCommand* cmd)
{
    boost::mutex::scoped_lock look(m_mutex);
    cmdque.push_back(cmd);
}
AsyncCommand* AsyncControlImpl::getCmd()
{
    AsyncCommand* cmd = NULL;
    boost::mutex::scoped_lock look(m_mutex);
    if (!cmdque.empty()) {
        cmd = cmdque.front();
        cmdque.pop_front();
    }
    return cmd;
}
