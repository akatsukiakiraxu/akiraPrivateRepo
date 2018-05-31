#ifndef SENSOR_H_INC_
#define SENSOR_H_INC_

#include <stddef.h>

#define NUM_CHANNEL 4
#define FIFO_SIZE   ((8*1024))
#define FFT_FIFO_SIZE   ((16*1024))

void start_sensor(
	unsigned long threshold_cnt,
	unsigned long threshold,
	void (*trigger_handler)(int code, int palette, int pos, int overflow),
	void (*data_handler)(const void *data, size_t length, const void *fft_data, size_t fft_length)
);

#endif /* SENSOR_H_INC_ */
