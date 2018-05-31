#define MAX_FILE 32
#define QUEUE_SIZE 256

#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h> /* For fsync */

#include "async-io.h"

static pthread_t pt;
static pthread_mutex_t mutex = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t cv = PTHREAD_COND_INITIALIZER;

enum command_id {
	OPEN,
	CLOSE,
	WRITE,
	REWIND_AND_WRITE,
	FINALIZE
};
struct command {
	enum command_id com_id;
	int file_id;
	size_t n;
	bool sync;
	void *buf;
};

static FILE *fps[MAX_FILE];
uint32_t used_list;

static struct command command_queue[QUEUE_SIZE];
volatile static size_t head;
volatile static size_t tail;

static bool is_empty(void)
{
	return (head == tail);
}

static bool is_fill(void)
{
	return (head + 1) % QUEUE_SIZE == tail;
}

static void enq_command(const struct command *com)
{
	bool emp;
	pthread_mutex_lock(&mutex);
	while (is_fill()) {
		pthread_cond_wait(&cv, &mutex);
	}
	command_queue[head] = *com;
	emp = is_empty();
	head = (head + 1) % QUEUE_SIZE;
	if (emp) {
		pthread_cond_signal(&cv);
	}
	pthread_mutex_unlock(&mutex);
}

static struct command *deq_command(void)
{
	bool fil;
	struct command *com;
	pthread_mutex_lock(&mutex);
	while (is_empty()) {
		pthread_cond_wait(&cv, &mutex);
	}
	com = &command_queue[tail];
	fil = is_fill();
	tail = (tail + 1) % QUEUE_SIZE;
	if (fil) {
		pthread_cond_signal(&cv);
	}
	pthread_mutex_unlock(&mutex);

	return com;
}

static int is_free(int n)
{
	return !((used_list >> n) & 0x1);
}

static void set_used(int n)
{
	used_list |= (uint32_t)0x1u << n;
}

static void set_freed(int n)
{
	used_list &= ~((uint32_t)0x1u << n);
}

static void *async_file_thread(void *arg)
{
	bool terminated = false;
	while (!terminated) {
		struct command *com = deq_command();
		switch (com->com_id) {
		case OPEN:
			fps[com->file_id] = fopen(com->buf, "wb");
			free(com->buf);
			break;
		case CLOSE:
			fclose(fps[com->file_id]);
			pthread_mutex_lock(&mutex);
			set_freed(com->file_id);
			pthread_mutex_unlock(&mutex);
			break;
		case REWIND_AND_WRITE:
			rewind(fps[com->file_id]);
			/* Fall Through */
		case WRITE:
			fwrite(com->buf, 1, com->n, fps[com->file_id]);
			free(com->buf);
			if (com->sync) {
				fflush(fps[com->file_id]);
#ifdef _POSIX_C_SOURCE
				fsync(fileno(fps[com->file_id]));
#else
                            fsync(fps[com->file_id]->_fileno);
#endif
			}
			break;
		case FINALIZE:
			terminated = true;
			break;
		}
	}
	return NULL;
}

void async_file_init(void)
{
	head = tail = 0;
	pthread_create(&pt, NULL, &async_file_thread, NULL);
}

void async_file_finalize(void)
{
	struct command com;
	com.com_id = FINALIZE;
	enq_command(&com);
	pthread_join(pt, NULL);
}

/* async files are always "wb" mode. */
int async_file_open(const char *path)
{
	struct command com;
	int fid = 0;

	pthread_mutex_lock(&mutex);
	while (!is_free(fid)) {
		if (fid >= MAX_FILE) {
			fid = -1;
			break;
		}
		fid++;
	}
	if (fid >= 0) {
		set_used(fid);
	}
	pthread_mutex_unlock(&mutex);

	if (fid >= 0) {
		com.buf = malloc(strlen(path) + 1);
		strcpy((char *)com.buf, path);

		com.com_id = OPEN;
		com.file_id = fid;
		enq_command(&com);
	}

	return fid;
}

void async_file_close(int file_no)
{
	struct command com;
	com.com_id = CLOSE;
	com.file_id = file_no;
	enq_command(&com);
}

void *async_file_malloc(size_t n)
{
	return malloc(n);
}

void async_file_free(void *ptr)
{
	free(ptr);
}

/* ptr bust be allocated by async_file_malloc and will be freed after writing. */
void async_file_write(int file_id, void *ptr, size_t n)
{
	struct command com;
	com.com_id = WRITE;
	com.file_id = file_id;
	com.n = n;
	com.sync = false;
	com.buf = ptr;
	enq_command(&com);
}

void async_file_write_and_sync(int file_id, void *ptr, size_t n)
{
	struct command com;
	com.com_id = WRITE;
	com.file_id = file_id;
	com.n = n;
	com.sync = true;
	com.buf = ptr;
	enq_command(&com);
}

void async_file_rewind_and_write(int file_id, void *ptr, size_t n)
{
	struct command com;
	com.com_id = REWIND_AND_WRITE;
	com.file_id = file_id;
	com.n = n;
	com.sync = false;
	com.buf = ptr;
	enq_command(&com);
}
