#ifndef ASYNC_IO_H_INC_
#define ASYNC_IO_H_INC_

#include <stddef.h>

void async_file_init(void);
void async_file_finalize(void);
int async_file_open(const char *path);
void async_file_close(int file_no);
void *async_file_malloc(size_t n);
void async_file_free(void *ptr);
void async_file_write(int file_id, void *ptr, size_t n);
void async_file_write_and_sync(int file_id, void *ptr, size_t n);
void async_file_rewind_and_write(int file_id, void *ptr, size_t n);


#endif /* ASYNC_IO_H_INC_ */
