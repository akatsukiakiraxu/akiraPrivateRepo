#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>

#include <fcntl.h>
#include <netdb.h>
#include <netinet/in.h>
#include <pthread.h>
#include <semaphore.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/wait.h>
#include <unistd.h>

#include "sensor.h"

#define AXI_XADC_IOCTL_BASE                     'W'
#define AXI_XADC_GET_NUM_DEVICES                _IO(AXI_XADC_IOCTL_BASE, 0)
#define AXI_XADC_GET_DEV_INFO                   _IO(AXI_XADC_IOCTL_BASE, 1)
#define AXI_XADC_DMA_CONFIG                     _IO(AXI_XADC_IOCTL_BASE, 2)
#define AXI_XADC_DMA_START                      _IO(AXI_XADC_IOCTL_BASE, 3)
#define AXI_XADC_DMA_STOP                       _IO(AXI_XADC_IOCTL_BASE, 4)
#define AXI_AD7476_CONFIG                       _IO(AXI_XADC_IOCTL_BASE, 5)
#define AXI_XADC_DMA_RESET                      _IO(AXI_XADC_IOCTL_BASE, 6)
#define ADC_THRESHOLD_SETUP0                    _IO(AXI_XADC_IOCTL_BASE, 7)
#define ADC_THRESHOLD_SETUP1                    _IO(AXI_XADC_IOCTL_BASE, 8)
#define ADC_TRIGGER                             _IO(AXI_XADC_IOCTL_BASE, 9)
#define ADC_RESET                       _IO(AXI_XADC_IOCTL_BASE, 10)
#define AXI_FFT_DMA_START		_IO(AXI_XADC_IOCTL_BASE, 11)
#define AXI_FFT_DMA_STOP		_IO(AXI_XADC_IOCTL_BASE, 12)
#define AXI_FFT_DMA_RESET		_IO(AXI_XADC_IOCTL_BASE, 12)

#define CONNECTION_EST_REG  0x00
#define RAW_DATA_REG        0x01        //- Raw data collection
#define XST_FAILURE -1
#define XST_SUCCESS  0
#define UART_BUF_SIZE 20

#define BAUDRATE B115200
#define DEV_NODE "/dev/ttyPS0"
#define _POSIX_SOURCE 1 /* POSIX compliant source */

/* #define DUMMY_TRIGGER 1 */

static int trigger_tri(uint32_t trigger)
{
	return trigger & 0x1;
}

static int trigger_code(uint32_t trigger)
{
	return (trigger >> 1) & 0x03;
}

static int trigger_palette(uint32_t trigger)
{
	return (trigger >> 3) & 0x1f;
}

static int trigger_pos(uint32_t trigger)
{
	return (trigger >> 8) & 0x03;
}

static int trigger_overflow(uint32_t trigger)
{
	return (trigger >> 31);
}

#define BUF_SIZE (4 * 1024 * 1024 / FIFO_SIZE)

struct sensor_out {
	uint32_t trigger;
	size_t length;
	size_t fft_length;
	char data[FIFO_SIZE];
	char fft_data[FFT_FIFO_SIZE];
};

static struct sensor_queue {
	pthread_mutex_t mutex;
	pthread_cond_t cv;

	struct sensor_out sensor_buf[BUF_SIZE];
	size_t head;
	bool on_enq;
	size_t tail;
	bool on_deq;
} sensor_queue = { PTHREAD_MUTEX_INITIALIZER, PTHREAD_COND_INITIALIZER };

static bool is_empty(struct sensor_queue *q)
{
	return (q->head == q->tail) ||
	       (q->head == (q->tail + 1) % BUF_SIZE && q->on_enq);
}

static bool is_fill(struct sensor_queue *q)
{
	return ((q->head + 1) % BUF_SIZE == q->tail) ||
	       ((q->head + 2) % BUF_SIZE == q->tail && q->on_deq);
}

struct sensor_out *start_enq(struct sensor_queue *q)
{
	struct sensor_out *val;
	pthread_mutex_lock(&q->mutex);
	while (is_fill(q)) {
		pthread_cond_wait(&q->cv, &q->mutex);
	}
	q->on_enq = true;
	val = &q->sensor_buf[q->head];
	q->head = (q->head + 1) % BUF_SIZE;
	pthread_mutex_unlock(&q->mutex);
	return val;
}

void finish_enq(struct sensor_queue *q)
{
	pthread_mutex_lock(&q->mutex);
	q->on_enq = false;
	pthread_cond_signal(&q->cv);
	pthread_mutex_unlock(&q->mutex);
}

struct sensor_out *start_deq(struct sensor_queue *q)
{
	struct sensor_out *val;
	pthread_mutex_lock(&q->mutex);
	while (is_empty(q)) {
		pthread_cond_wait(&q->cv, &q->mutex);
	}
	q->on_deq = true;
	val = &q->sensor_buf[q->tail];
	q->tail = (q->tail + 1) % BUF_SIZE;
	pthread_mutex_unlock(&q->mutex);
	return val;
}

void finish_deq(struct sensor_queue *q)
{
	pthread_mutex_lock(&q->mutex);
	q->on_deq = false;
	pthread_cond_signal(&q->cv);
	pthread_mutex_unlock(&q->mutex);
}

struct adc_args {
	unsigned long threshold_cnt;
	unsigned long threshold;
};


static void *sensor_thread(void *args_v)
{
	int adc_fd = open("/dev/channel0", O_RDONLY);
	int fft_fd = open("/dev/fft", O_RDONLY);
	char *data0;
	int r;
	// uint32_t trigger;
	int stop = 0;
	ssize_t length;
	struct sensor_out *sout;
	struct adc_args *args = (struct adc_args *)args_v;

	data0 = malloc(FIFO_SIZE);

	r = ioctl(adc_fd, AXI_AD7476_CONFIG);
	if (r != XST_SUCCESS) {
		perror("AD7476 config");
		return NULL;
	}

	r = ioctl(adc_fd, ADC_THRESHOLD_SETUP0, &args->threshold_cnt);
	if (r != XST_SUCCESS) {
		perror("Trigger setup");
		return NULL;
	}

	r = ioctl(adc_fd, ADC_THRESHOLD_SETUP1, &args->threshold);
	if (r != XST_SUCCESS) {
		perror("Trigger setup");
		return NULL;
	}

	while (!stop) {
		r = ioctl(adc_fd, AXI_XADC_DMA_START);
		if (r != XST_SUCCESS) {
			perror("dma start");
			return NULL;
		}
		r = ioctl(fft_fd, AXI_FFT_DMA_START);
		if (r != XST_SUCCESS) {
			perror("dma start");
			return NULL;
		}

		sout = start_enq(&sensor_queue);

		sout->fft_length = length = read(fft_fd, sout->fft_data, FFT_FIFO_SIZE);
		if (length <= 0) {
			perror("fft read");
			return NULL;
		}

		sout->length = length = read(adc_fd, sout->data, FIFO_SIZE);
		if (length <= 0) {
			perror("read");
			return NULL;
		}

		r = ioctl(adc_fd, AXI_XADC_DMA_STOP, &sout->trigger);
		if (r != XST_SUCCESS) {
			perror("dma stop");
			return NULL;
		}
		r = ioctl(fft_fd, AXI_FFT_DMA_STOP);
		if (r != XST_SUCCESS) {
			perror("dma stop");
			return NULL;
		}

		finish_enq(&sensor_queue);
	}

	r = ioctl(adc_fd, AXI_XADC_DMA_RESET);
	if (r != XST_SUCCESS) {
		perror("dma stop");
		return NULL;
	}

	free(data0);
	close(adc_fd);

	return NULL;
}

void start_sensor(
	unsigned long threshold_cnt,
	unsigned long threshold,
	void (*trigger_handler)(int code, int palette, int pos, int overflow),
	void (*data_handler)(const void *data, size_t length, const void *fft_data, size_t fft_length)
)
{
	struct adc_args args = { threshold_cnt, threshold };
	struct sensor_out *sout;

	pthread_t pt;

#ifdef DUMMY_TRIGGER
	time_t last_trigger = 0;
	time_t cur;
	static uint16_t hdata[FIFO_SIZE / sizeof(uint16_t)];
	static uint16_t ldata[FIFO_SIZE / sizeof(uint16_t)];
	size_t i;

	for (i = 0; i < FIFO_SIZE / sizeof(uint16_t); ++i) {
		hdata[i] = 4000;
		ldata[i] = 500;
	}
	last_trigger = time(NULL);
#endif

	pthread_create(&pt, NULL, &sensor_thread, &args);
	while (1) {
		sout = start_deq(&sensor_queue);
#ifdef DUMMY_TRIGGER
		cur = time(NULL);
		if (last_trigger + 3 <= cur) {
			last_trigger = cur;
			trigger_handler(0, 0, 0, 0);
			// data_handler(hdata, FIFO_SIZE);
			// for (i = 0; i < FIFO_SIZE; ++i) {
			// 	sout->data[i] = 0x80;
			// }
		} else {
			// data_handler(ldata, FIFO_SIZE);
		}
		data_handler(sout->data, sout->length, sout->fft_data, sout->fft_length);
#else
		if (trigger_handler != NULL &&
		    trigger_tri(sout->trigger)) {
			trigger_handler(
				trigger_code(sout->trigger),
				trigger_palette(sout->trigger),
				trigger_pos(sout->trigger),
				trigger_overflow(sout->trigger)
			);
		}
		data_handler(sout->data, sout->length, sout->fft_data, sout->fft_length);
#endif
		finish_deq(&sensor_queue);
	}
}
