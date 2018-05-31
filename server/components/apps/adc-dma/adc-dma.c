#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <string.h>
#include <stdint.h>
#include <fcntl.h>
#include <sys/time.h>
#include <time.h>
#include "adc-dma.h"
#include <sys/stat.h>
#include <pthread.h>
#include <semaphore.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netdb.h>
#include <sys/wait.h>

#define	DROP_CACHE		/* Enable drop page cache */

int stop;

#ifdef USE_SOCKET
#define PORT 23456

typedef struct {
	char* hostname;
	sem_t* sem_sock;
} sockthread_t;

pthread_t sock_pt;
sockthread_t sockthread_args;
sem_t sem_sock;

void* sock_thread(void* args)
{
	sockthread_t* pt = (sockthread_t *)args;
	sem_t* sem_sock =  pt->sem_sock;
	int s;
	struct sockaddr_in addr;
	struct hostent* hp;
	char buf[32];

	while (1) {
		if ((s=socket(AF_INET, SOCK_STREAM, 0)) < 0) {
			perror("socket failed");
			sleep(1);
			continue;
		}

		memset((char*)&addr, 0, sizeof(addr));
		if ((hp=gethostbyname(pt->hostname)) == NULL) {
			perror("gethostbyname failed");
			sleep(1);
			close(s);
			continue;
		}
		bcopy(hp->h_addr, &addr.sin_addr, hp->h_length);
		addr.sin_family = AF_INET;
		addr.sin_port = htons(PORT);

		if (connect(s, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
			perror("sock connect error");
			sleep(1);
			close(s);
			continue;
		}

		while(1) {
			memset(buf, 0, sizeof(buf));
			int r = read(s, buf, sizeof(buf));
			if (r <= 0) {
				// EOF or error
				break;
			}
			if (strcmp(buf, "START") == 0) {
				fprintf(stderr, "start\n");
				sem_post(sem_sock);
			} else if (strcmp(buf, "STOP") == 0) {
				fprintf(stderr, "stop\n");
				stop = 1;
			}
			usleep(1000*500);
		}
		close(s);
	}

	return NULL;
}

void sock_thread_create(char *hostname)
{
	sem_init(&sem_sock, 0, 0);
	sockthread_args.sem_sock = &sem_sock;
	sockthread_args.hostname = hostname;

	pthread_create(&sock_pt, NULL, &sock_thread, &sockthread_args);
	//sem_wait(&sem_sock);
}

void sock_thread_terminate(void)
{
	pthread_cancel(sock_pt);
	pthread_join(sock_pt, NULL);
	sem_destroy(&sem_sock);
}
#endif /*USE_SOCKET*/

typedef struct {
	FILE* next_fp;
	FILE* prev_fp;
	sem_t* sem;
	int stop;
} fopenthread_t;

#define	DROP_CACHE_MSG_1	"1\n"	/* Purge the page cache of file data */
#define	DROP_CACHE_MSG_4	"4\n"	/* Disable logging of drop cache */
#define	DROP_CACHE_MSG_LEN	2

void* fopen_thread(void* args)
{
#ifdef DROP_CACHE
	int drop_cache_fd;
#endif
	int suffix = 0;
	char time_str[64];
	char filename[255];
	struct timeval myTime;
	struct tm* local_time;

	fopenthread_t* pt = (fopenthread_t*)args;
	sem_t* sem =  pt->sem;

   	setenv("TZ", "JST-9", 1);
   	tzset();

        gettimeofday(&myTime, NULL);
	local_time = localtime(&myTime.tv_sec);
	sprintf(time_str, "%04d%02d%02d-%02d%02d%02d.%06d",
		local_time->tm_year + 1900, local_time->tm_mon + 1, local_time->tm_mday,
		local_time->tm_hour, local_time->tm_min, local_time->tm_sec,
		myTime.tv_usec);
#ifdef DROP_CACHE
	drop_cache_fd = open("/proc/sys/vm/drop_caches", O_WRONLY | O_TRUNC);
	if (drop_cache_fd > 0) {
		lseek(drop_cache_fd, 0, SEEK_SET);
		write(drop_cache_fd, DROP_CACHE_MSG_4, DROP_CACHE_MSG_LEN);
	} else
		perror("open(drop_cache)");
#endif
	while (1) {
		sem_wait(sem);
		if (pt->prev_fp != NULL) {
#ifdef DROP_CACHE
			fflush(pt->prev_fp);
			fdatasync(fileno(pt->prev_fp));
			if (drop_cache_fd > 0) {
				lseek(drop_cache_fd, 0, SEEK_SET);
				write(drop_cache_fd, DROP_CACHE_MSG_1, DROP_CACHE_MSG_LEN);
			}
#endif
			fclose(pt->prev_fp);
			pt->prev_fp = NULL;
		}
		if (pt->stop != 0)
			break;
		sprintf(filename, "%s.%04d", time_str, suffix);
		FILE* fp = fopen(filename, "w+");
		if (fp == NULL) {
			fprintf(stderr, "file %s open failed\n", filename);
			pt->stop = 1;
			break;
		}
		pt->next_fp = fp;
		__sync_synchronize();
		++suffix;
	}
#ifdef DROP_CACHE
	if (drop_cache_fd > 0)
		close(drop_cache_fd);
#endif
	return NULL;
}

void write_summary(FILE* fp, int r, const void* read_buf)
{
        static uint32_t sum = 0, acc_count = 0, max_data = 0, min_data=0xffffffff;
	const uint16_t* read_buf_u16 = (const uint16_t *)read_buf;
	const int n = r / sizeof(uint16_t);
        max_data = read_buf_u16[0];
        min_data = read_buf_u16[0];

	for (int i = 0; i < n; ++i) {
                  //max
                if(max_data < read_buf_u16[i]){
                     max_data = read_buf_u16[i];
                }
                //min
                if(min_data > read_buf_u16[i] && read_buf_u16[i] != 0 ){
                     min_data = read_buf_u16[i];
                }

		//sum += read_buf_u16[i];
		++acc_count;
                //FIFO_SIZE:4096
		//4096/1=244 sampling
		//4096/4=1K samping
		//if (acc_count == FIFO_SIZE/4) {
		//if (acc_count == FIFO_SIZE/16) {
		if (acc_count == FIFO_SIZE) {
			//const uint32_t average = sum / acc_count;
			const uint8_t out_buf[sizeof(uint32_t)] = {
				// Big endian
				//(uint8_t)((average >> 24) & 0xffu),
				//(uint8_t)((average >> 16) & 0xffu),
				//(uint8_t)((average >>  8) & 0xffu),
				//(uint8_t)((average >>  0) & 0xffu),
                                (uint8_t)((max_data >>  8) & 0xffu),
                                (uint8_t)((max_data >>  0) & 0xffu),
                                (uint8_t)((min_data >>  8) & 0xffu),
                                (uint8_t)((min_data >>  0) & 0xffu),

			};
			fwrite(out_buf, sizeof(out_buf), 1, fp);
			fflush(fp);
			sum = acc_count = 0;
		}
	}
}

#ifdef USE_PIPE
int file_write(int pipe_fd0)
{
	fopenthread_t fopenthread_args;
	sem_t fopen_sem;
	pthread_t fopen_pt;

	FILE* fd = NULL;
	char* read_buf = NULL;
	long long written = 0;

	int retcode = 0;
	int trigger = 0;

	read_buf = malloc(FIFO_SIZE+4);
	if (read_buf == NULL) {
		fprintf(stderr, "create buffer failed\n");
		return 1;
	}
	memset(read_buf, 0, FIFO_SIZE+4);

	sem_init(&fopen_sem, 0, 1u);
	fopenthread_args.next_fp = NULL;
	fopenthread_args.prev_fp = NULL;
	fopenthread_args.sem = &fopen_sem;
	fopenthread_args.stop = 0;
	pthread_create(&fopen_pt, NULL, &fopen_thread, &fopenthread_args);

	for (;;) {
		if (written >= 512 * 1024 * 1024 || trigger == 1) {
		//if (written >= 20 * 1024 * 1024 || trigger == 1) {  //10sec
		//if (written >= 240 * 1024 * 1024 || trigger == 1) { //2 min
			while (1) {
				const int done = __sync_bool_compare_and_swap(
					(void **)&fopenthread_args.prev_fp,
					NULL, (void *)fd);
				if (done)
					break;
			}
			fd = NULL;
			written = 0;
			trigger = 0;
		}
		if (fd == NULL) {
			do {
				__sync_synchronize();
				if (fopenthread_args.stop) {
					retcode = 2;
					goto exit;
				}
				fd = fopenthread_args.next_fp;
			} while (fd == NULL);
			fopenthread_args.next_fp = NULL;
			sem_post(&fopen_sem);
		}

		const int r = read(pipe_fd0, read_buf, FIFO_SIZE + 4);
		if (r == 0)	// EOF
			break;
		if (r < 0) {
			perror("read");
			retcode = 1;
			goto exit;
		}
	        trigger = read_buf[FIFO_SIZE]; 	
		fwrite(read_buf, r-4, 1, fd);
		written += r;

		write_summary(stdout, r, read_buf);
                //printf("read pipe %d ok\n",trigger );
	}
exit:
	fopenthread_args.stop = 1;
	sem_post(&fopen_sem);
	pthread_join(fopen_pt, NULL);
	sem_destroy(&fopen_sem);
	if (fopenthread_args.next_fp != NULL)
		fclose(fopenthread_args.next_fp);

	if (fd != NULL)
		fclose(fd);
	free(read_buf);

	return retcode;
}
#else
struct file_write_data {
	fopenthread_t fopenthread_args;
	sem_t fopen_sem;
	pthread_t fopen_pt;
	FILE* fd;
	char* read_bu;
	long long written;
	int trigger;
};

void file_write_init(struct file_write_data *fwd)
{
	sem_init(&fwd->fopen_sem, 0, 1u);
	fwd->fopenthread_args.next_fp = NULL;
	fwd->fopenthread_args.prev_fp = NULL;
	fwd->fopenthread_args.sem = &fwd->fopen_sem;
	fwd->fopenthread_args.stop = 0;
	pthread_create(&fwd->fopen_pt, NULL, fopen_thread, &fwd->fopenthread_args);
}

void file_write_fin(struct file_write_data *fwd)
{
	fwd->fopenthread_args.stop = 1;
	sem_post(&fwd->fopen_sem);
	pthread_join(fwd->fopen_pt, NULL);
	sem_destroy(&fwd->fopen_sem);
	if (fwd->fopenthread_args.next_fp != NULL)
		fclose(fwd->fopenthread_args.next_fp);
	if (fwd->fd != NULL)
		fclose(fwd->fd);
}

int file_write(char *read_buf, int rs, struct file_write_data *fwd)
{
	//if (fwd->written >= 20 * 1024 * 1024 || fwd->trigger == 1) {  //10sec
	if (fwd->written >= 512 * 1024 * 1024 || fwd->trigger == 1) {  //10sec
		while (1) {
			const int done = __sync_bool_compare_and_swap(
				(void **)&fwd->fopenthread_args.prev_fp,
				    NULL, (void *)fwd->fd);
			if (done)
				break;
		}
		fwd->fd = NULL;
		fwd->written = 0;
		fwd->trigger = 0;
	}
	if (fwd->fd == NULL) {
		do {
			__sync_synchronize();
			if (fwd->fopenthread_args.stop) {
				return -1;
			}
			fwd->fd = fwd->fopenthread_args.next_fp;
		} while (fwd->fd == NULL);
		fwd->fopenthread_args.next_fp = NULL;
		sem_post(&fwd->fopen_sem);
	}

	fwd->trigger = read_buf[FIFO_SIZE];
	fwrite(read_buf, rs, 1, fwd->fd);
	fwd->written += rs;

	write_summary(stdout, rs, read_buf);
	return 0;
}
#endif

#ifdef USE_PIPE
int dma_read(char *hostname, char *threshold_cnt_c, char *threshold_c, int pipe_fd1)
#else
int dma_read(char *hostname, char *threshold_cnt_c, char *threshold_c,
    struct file_write_data *fwd)
#endif
{
	int adc_fd = open("/dev/channel0", O_RDONLY);
	char *data0;
	int r, rs, trigger,reset;
#ifdef USE_PIPE
	int s, ws;
#endif
	long long num_sample_total = 0;

#ifdef USE_SOCKET
	sock_thread_create(hostname);
#endif
	stop = 0;
	data0 = malloc(FIFO_SIZE+4);
	if (data0 == NULL) {
		fprintf(stderr, "create buffer failed\n");
		return 1;
	}
	memset(data0, 0, FIFO_SIZE+4);

	r = ioctl(adc_fd, AXI_AD7476_CONFIG);
	if (r != XST_SUCCESS) {
		perror("AD7476 config");
		return 1;
	}
        unsigned long threshold_cnt, threshold;  
        threshold_cnt=0x200;
        threshold = 0x400;
         reset = 0x0;
        threshold_cnt = atoi(threshold_cnt_c);
        threshold = atoi(threshold_c);

        r = ioctl(adc_fd, ADC_THRESHOLD_SETUP0, &threshold_cnt);
        if (r != XST_SUCCESS) {
                perror("Trigger setup");
                return 1;
        }
        r = ioctl(adc_fd, ADC_THRESHOLD_SETUP1, &threshold);
        if (r != XST_SUCCESS) {
                perror("Trigger setup");
                return 1;
        }
        //r = ioctl(adc_fd, ADC_RESET, &reset);
        //if (r != XST_SUCCESS) {
        //        perror("set RESET");
        //        return 1;
        //}


	while (1) {
		r = ioctl(adc_fd, AXI_XADC_DMA_START);
		if (r != XST_SUCCESS) {
			perror("dma start");
			return 1;
		}

		rs = read(adc_fd, data0, FIFO_SIZE);
		if (rs <= 0) {
			perror("read");
			return 1;
		}
                r = ioctl(adc_fd, AXI_XADC_DMA_STOP, &trigger);
                if (r != XST_SUCCESS) {
			perror("dma stop");
			return 1;
                }
		if(trigger == 1) {
	        	//printf("Trigger %x ok\n",trigger );
		}
    		data0[FIFO_SIZE]=trigger;
                //printf("write pipe %x ok\n",data0[FIFO_SIZE] );
#ifdef USE_PIPE
		ws = 0;
		do {
			s = write(pipe_fd1, data0 + ws, rs+4);
			if (s == 0) {	// pipe closed.
			  stop = 1;
			  break;
			}
			if (s < 0) {
				perror("write");
				return 1;
			}
			ws += s;
			rs -= s;
		} while (ws < r);
#else
		if (file_write(data0, rs, fwd) < 0)
			stop = 1;
#endif
		num_sample_total += r / 2;
		if (stop == 1)
			break;
	}

	//r = ioctl(adc_fd, AXI_AD7476_CONFIG);
	//if (r != XST_SUCCESS) {
	//	perror("AD7476 config");
	//	return 1;
	//}
	r = ioctl(adc_fd, AXI_XADC_DMA_RESET);
	if (r != XST_SUCCESS) {
		perror("dma stop");
		return 1;
	}

#ifdef USE_SOCKET
	sock_thread_terminate();
#endif
	fprintf(stderr, "%lld\n", num_sample_total);
	free(data0);
	return 0;
}

#ifdef USE_PIPE
int main(int argc, char** argv)
{
	while (1) {
		int pipe_fd[2];
		int pid;
		int r, s;

		if (pipe(pipe_fd) < 0) {
			perror("pipe");
			return 1;
		}

		pid = fork();
		if (pid < 0) {
			perror("fork");
			return 1;
		}

		if (pid == 0) {
			close(pipe_fd[1]);
			r = file_write(pipe_fd[0]);
			close(pipe_fd[0]);
			exit(r);
		} else {
			close(pipe_fd[0]);
			r = dma_read(argv[2], argv[3], argv[4], pipe_fd[1]);
			close(pipe_fd[1]);
			wait(&s);
			s = WEXITSTATUS(s);
		}
	}
	return 0;
}
#else
int main(int argc, char** argv)
{
	struct file_write_data fwd;
	memset(&fwd, 0, sizeof(fwd));

	file_write_init(&fwd);
	dma_read(argv[2], argv[3], argv[4], &fwd);
	file_write_fin(&fwd);
}
#endif
