#include <stdio.h>
#include <stdlib.h>

#include "sensor.h"

#define NUM_CODE     32
#define NUM_PALETTE 128
#define DATA_LENGTH (8 * 1024 * 1024)

void start_sensor(
	unsigned long threshold_cnt,
	unsigned long threshold,
	void (*handler)(int code, int palette, const void *data, size_t length, int trigger)
)
{
	int code, palette;
	char *data;

	data = malloc(DATA_LENGTH);
	if (data == NULL) {
		perror("Memory allocation error.");
		return;
	}

	while (1) {
		for (code = 0; code < NUM_CODE; ++code) {
			for (palette = 0; palette < NUM_PALETTE; ++palette) {
				handler(code, palette, data, DATA_LENGTH, 1);
			}
		}
	}

	free(data);
}
