#ifndef _DEBUG_H_
#define _DEBUG_H_
#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <stdarg.h>
/*
inline void ADCDB_DEBUG(bool flag, char* format, ... ) {
    if (flag) {
        char buf[256] = {'\0'};
        va_list arg;
        va_start(arg, format);
        vsprintf(buf, format, arg);
        va_end(arg);
        fprintf(stderr, "%s\n", buf);
    }
    else {
    }
}
*/
#define STRINGIFY(n) #n
#define TOSTRING(n) STRINGIFY(n)
#define PREFIX __FILE__ ":" TOSTRING(__LINE__) ": "

#define log(fp, prefix, ...)                    \
  do {                                          \
    fprintf(fp, prefix " " PREFIX __VA_ARGS__); \
    fprintf(fp, "\n");                          \
  } while (0)

#define ADCDB_DEBUG(flag, args...)              \
    do {                                        \
        if (flag)                               \
            log(stderr, "[d]", args);           \
    } while (0)

#define ADCDB_INFO(args...) log(stderr, "[i]", args)
#define ADCDB_ERR(args...) log(stderr, "[e]", args)

#endif /* _DEBUG_H_ */
